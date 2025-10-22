//go:build linux

package sessions

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type commandRunner interface {
	CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

// NewManager constructs a logind-backed session manager.
func NewManager() (Manager, error) {
	if _, err := exec.LookPath("loginctl"); err != nil {
		return nil, fmt.Errorf("loginctl not found: %w", err)
	}
	mgr := &logindManager{
		runner:   execRunner{},
		interval: 5 * time.Second,
		stop:     make(chan struct{}),
	}
	return mgr, nil
}

type logindManager struct {
	runner   commandRunner
	interval time.Duration

	mu     sync.Mutex
	closed bool
	stop   chan struct{}
}

func (m *logindManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	close(m.stop)
	m.closed = true
	return nil
}

func (m *logindManager) List() ([]Session, error) {
	sessions, err := m.enumerate(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]Session, 0, len(sessions))
	for _, sess := range sessions {
		out = append(out, sess)
	}
	return out, nil
}

func (m *logindManager) Watch(stop <-chan struct{}) (<-chan Event, error) {
	events := make(chan Event, 8)
	go m.monitor(stop, events)
	return events, nil
}

func (m *logindManager) monitor(stop <-chan struct{}, events chan<- Event) {
	defer close(events)

	ctx := context.Background()
	previous := map[string]Session{}

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		default:
		}

		sessions, err := m.enumerate(ctx)
		if err != nil {
			select {
			case events <- Event{Type: EventError, Err: err}:
			case <-stop:
				return
			}
		} else {
			m.emitDiff(events, stop, previous, sessions)
			previous = sessions
		}

		select {
		case <-stop:
			return
		case <-ticker.C:
		}
	}
}

func (m *logindManager) emitDiff(events chan<- Event, stop <-chan struct{}, previous, current map[string]Session) {
	for id, sess := range current {
		old, ok := previous[id]
		typ := EventAdded
		if ok {
			if sess.Equal(old) {
				continue
			}
			typ = EventUpdated
		}
		select {
		case events <- Event{Type: typ, Session: sess}:
		case <-stop:
			return
		}
	}

	for id, sess := range previous {
		if _, ok := current[id]; ok {
			continue
		}
		select {
		case events <- Event{Type: EventRemoved, Session: sess}:
		case <-stop:
			return
		}
	}
}

func (m *logindManager) enumerate(ctx context.Context) (map[string]Session, error) {
	raw, err := m.runner.CombinedOutput(ctx, "loginctl", "list-sessions", "--no-legend")
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	sessions := make(map[string]Session)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		id := fields[0]
		props, err := m.sessionProperties(ctx, id)
		if err != nil {
			log.Printf("logind: session %s properties error: %v", id, err)
			continue
		}
		if !strings.EqualFold(props["Active"], "yes") {
			continue
		}
		if strings.EqualFold(props["Remote"], "yes") {
			continue
		}
		uid, err := strconv.ParseUint(props["UID"], 10, 32)
		if err != nil {
			continue
		}
		gid, err := strconv.ParseUint(props["GID"], 10, 32)
		if err != nil {
			continue
		}
		runtimeDir := props["RuntimePath"]
		if runtimeDir == "" {
			runtimeDir = fmt.Sprintf("/run/user/%s", props["UID"])
		}
		env := map[string]string{
			"XDG_RUNTIME_DIR": runtimeDir,
		}
		if display := props["Display"]; display != "" {
			env["DISPLAY"] = display
		}
		if _, ok := env["DISPLAY"]; !ok && props["Type"] == "x11" {
			if tty := props["TTY"]; tty != "" {
				env["DISPLAY"] = ":0"
			}
		}
		if _, ok := env["DBUS_SESSION_BUS_ADDRESS"]; !ok {
			env["DBUS_SESSION_BUS_ADDRESS"] = fmt.Sprintf("unix:path=%s/bus", runtimeDir)
		}
		sess := Session{
			ID:         id,
			User:       props["Name"],
			UID:        uint32(uid),
			GID:        uint32(gid),
			RuntimeDir: runtimeDir,
			Display:    env["DISPLAY"],
			Env:        env,
		}
		sessions[id] = sess
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (m *logindManager) sessionProperties(ctx context.Context, id string) (map[string]string, error) {
	raw, err := m.runner.CombinedOutput(ctx, "loginctl", "show-session", id, "-p", "Name", "-p", "UID", "-p", "GID", "-p", "Display", "-p", "Remote", "-p", "Active", "-p", "Type", "-p", "RuntimePath", "-p", "TTY")
	if err != nil {
		return nil, err
	}
	props := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		props[parts[0]] = strings.TrimSpace(parts[1])
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if props["Name"] == "" {
		return nil, errors.New("session name missing")
	}
	return props, nil
}
