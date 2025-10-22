package service

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/example/gotray/internal/service/sessions"
)

type mockManager struct {
	list   []sessions.Session
	events chan sessions.Event
}

func newMockManager(list []sessions.Session) *mockManager {
	return &mockManager{list: list, events: make(chan sessions.Event, 10)}
}

func (m *mockManager) List() ([]sessions.Session, error) {
	out := make([]sessions.Session, len(m.list))
	copy(out, m.list)
	return out, nil
}

func (m *mockManager) Watch(stop <-chan struct{}) (<-chan sessions.Event, error) {
	ch := make(chan sessions.Event)
	go func() {
		defer close(ch)
		for {
			select {
			case <-stop:
				return
			case ev, ok := <-m.events:
				if !ok {
					return
				}
				ch <- ev
			}
		}
	}()
	return ch, nil
}

func (m *mockManager) Close() error {
	close(m.events)
	return nil
}

type fakeProcess struct {
	mu        sync.Mutex
	done      chan struct{}
	err       error
	stopCount int
}

func newFakeProcess() *fakeProcess {
	return &fakeProcess{done: make(chan struct{})}
}

func (p *fakeProcess) Done() <-chan struct{} { return p.done }

func (p *fakeProcess) Err() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

func (p *fakeProcess) Stop() error {
	p.mu.Lock()
	p.stopCount++
	if p.err == nil {
		p.err = context.Canceled
	}
	if !channelClosed(p.done) {
		close(p.done)
	}
	p.mu.Unlock()
	return nil
}

func (p *fakeProcess) Exit(err error) {
	p.mu.Lock()
	p.err = err
	if !channelClosed(p.done) {
		close(p.done)
	}
	p.mu.Unlock()
}

func (p *fakeProcess) Stops() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stopCount
}

func channelClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestSupervisorLaunchesInitialSessions(t *testing.T) {
	t.Setenv("GOTRAY_CONFIG_PATH", filepath.Join(t.TempDir(), "config.enc"))
	sess := sessions.Session{ID: "1", User: "alice"}
	mgr := newMockManager([]sessions.Session{sess})
	defer mgr.Close()

	var mu sync.Mutex
	launches := 0
	processes := make(map[string]*fakeProcess)

	launcher := func(ctx context.Context, session sessions.Session, token, endpoint string) (agentProcess, error) {
		mu.Lock()
		defer mu.Unlock()
		launches++
		fp := newFakeProcess()
		processes[session.ID] = fp
		return fp, nil
	}

	svc, err := newService("secret", mgr, launcher)
	if err != nil {
		t.Fatalf("newService: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.startSupervisor(ctx)

	waitFor(t, func() bool { return svc.supervisor != nil })

	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return launches == 1
	})

	if _, ok := processes[sess.ID]; !ok {
		t.Fatalf("expected process for session %s", sess.ID)
	}
}

func TestSupervisorStopsOnSessionRemoval(t *testing.T) {
	t.Setenv("GOTRAY_CONFIG_PATH", filepath.Join(t.TempDir(), "config.enc"))
	sess := sessions.Session{ID: "2", User: "bob"}
	mgr := newMockManager([]sessions.Session{sess})
	defer mgr.Close()

	fp := newFakeProcess()
	var mu sync.Mutex
	started := false
	launcher := func(ctx context.Context, session sessions.Session, token, endpoint string) (agentProcess, error) {
		mu.Lock()
		started = true
		mu.Unlock()
		return fp, nil
	}

	svc, err := newService("secret", mgr, launcher)
	if err != nil {
		t.Fatalf("newService: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.startSupervisor(ctx)

	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return started
	})

	mgr.events <- sessions.Event{Type: sessions.EventRemoved, Session: sess}

	waitFor(t, func() bool { return fp.Stops() > 0 })
}

func TestSupervisorRestartsAfterFailure(t *testing.T) {
	t.Setenv("GOTRAY_CONFIG_PATH", filepath.Join(t.TempDir(), "config.enc"))
	sess := sessions.Session{ID: "3", User: "carol"}
	mgr := newMockManager([]sessions.Session{sess})
	defer mgr.Close()

	var mu sync.Mutex
	launches := 0
	current := newFakeProcess()

	launcher := func(ctx context.Context, session sessions.Session, token, endpoint string) (agentProcess, error) {
		mu.Lock()
		defer mu.Unlock()
		launches++
		if launches > 1 {
			current = newFakeProcess()
		}
		return current, nil
	}

	svc, err := newService("secret", mgr, launcher)
	if err != nil {
		t.Fatalf("newService: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.startSupervisor(ctx)

	waitFor(t, func() bool { return svc.supervisor != nil })

	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return launches == 1
	})

	svc.supervisor.mu.Lock()
	svc.supervisor.restartDelay = 20 * time.Millisecond
	svc.supervisor.mu.Unlock()

	current.Exit(context.DeadlineExceeded)

	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return launches >= 2
	})
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met before deadline")
}
