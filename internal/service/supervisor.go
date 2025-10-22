package service

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/example/gotray/internal/service/sessions"
)

type agentProcess interface {
	Done() <-chan struct{}
	Err() error
	Stop() error
}

type launchFunc func(context.Context, sessions.Session, string, string) (agentProcess, error)

type sessionSupervisor struct {
	ctx          context.Context
	cancel       context.CancelFunc
	mgr          sessions.Manager
	launch       launchFunc
	token        string
	endpoint     string
	restartDelay time.Duration

	mu     sync.Mutex
	agents map[string]*agentState
}

type agentState struct {
	session sessions.Session
	proc    agentProcess
	cancel  context.CancelFunc
}

func newSessionSupervisor(parent context.Context, mgr sessions.Manager, launcher launchFunc, token, endpoint string) *sessionSupervisor {
	ctx, cancel := context.WithCancel(parent)
	return &sessionSupervisor{
		ctx:          ctx,
		cancel:       cancel,
		mgr:          mgr,
		launch:       launcher,
		token:        token,
		endpoint:     endpoint,
		restartDelay: 2 * time.Second,
		agents:       make(map[string]*agentState),
	}
}

func (s *sessionSupervisor) run() {
	defer s.cancel()

	if sessions, err := s.mgr.List(); err != nil {
		log.Printf("service: unable to enumerate sessions: %v", err)
	} else {
		for _, sess := range sessions {
			s.start(sess)
		}
	}

	events, err := s.mgr.Watch(s.ctx.Done())
	if err != nil {
		log.Printf("service: session watch error: %v", err)
		return
	}

	for {
		select {
		case <-s.ctx.Done():
			s.stopAll()
			return
		case ev, ok := <-events:
			if !ok {
				s.stopAll()
				return
			}
			s.handleEvent(ev)
		}
	}
}

func (s *sessionSupervisor) handleEvent(ev sessions.Event) {
	switch ev.Type {
	case sessions.EventAdded, sessions.EventUpdated:
		s.start(ev.Session)
	case sessions.EventRemoved:
		s.stop(ev.Session.ID)
	case sessions.EventError:
		if ev.Err != nil {
			log.Printf("service: session watcher error: %v", ev.Err)
		}
	}
}

func (s *sessionSupervisor) start(sess sessions.Session) {
	s.mu.Lock()
	if s.ctx.Err() != nil {
		s.mu.Unlock()
		return
	}

	var replaced *agentState
	if existing, ok := s.agents[sess.ID]; ok {
		if existing.session.Equal(sess) {
			s.mu.Unlock()
			return
		}
		delete(s.agents, sess.ID)
		replaced = existing
	}
	s.mu.Unlock()

	if replaced != nil {
		replaced.cancel()
		if err := replaced.proc.Stop(); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("service: stop existing tray for %s: %v", sess.User, err)
		}
	}

	s.mu.Lock()
	if s.ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(s.ctx)
	proc, err := s.launch(ctx, sess, s.token, s.endpoint)
	if err != nil {
		s.mu.Unlock()
		cancel()
		log.Printf("service: launch tray for %s: %v", sess.User, err)
		return
	}

	state := &agentState{session: sess, proc: proc, cancel: cancel}
	s.agents[sess.ID] = state
	s.mu.Unlock()
	go s.monitor(sess.ID, state)
}

func (s *sessionSupervisor) stop(id string) {
	s.mu.Lock()
	state, ok := s.agents[id]
	if ok {
		delete(s.agents, id)
	}
	s.mu.Unlock()
	if !ok {
		return
	}
	state.cancel()
	if err := state.proc.Stop(); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("service: stop tray for %s: %v", state.session.User, err)
	}
}

func (s *sessionSupervisor) stopAll() {
	s.mu.Lock()
	states := make([]*agentState, 0, len(s.agents))
	for id, st := range s.agents {
		delete(s.agents, id)
		states = append(states, st)
	}
	s.mu.Unlock()

	for _, st := range states {
		st.cancel()
		if err := st.proc.Stop(); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("service: stop tray for %s: %v", st.session.User, err)
		}
	}
}

func (s *sessionSupervisor) monitor(id string, st *agentState) {
	<-st.proc.Done()
	err := st.proc.Err()

	s.mu.Lock()
	current, ok := s.agents[id]
	if ok && current == st {
		delete(s.agents, id)
	}
	s.mu.Unlock()

	st.cancel()

	if s.ctx.Err() != nil {
		return
	}

	if err != nil {
		log.Printf("service: tray for %s exited: %v", st.session.User, err)
		delay := s.restartDelay
		if delay <= 0 {
			delay = 2 * time.Second
		}
		time.AfterFunc(delay, func() {
			s.start(st.session)
		})
	}
}
