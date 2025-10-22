package sessions

import "errors"

// EventType represents the type of session change reported by the manager.
type EventType int

const (
	// EventAdded indicates a new interactive session is now available.
	EventAdded EventType = iota + 1
	// EventRemoved indicates an interactive session terminated.
	EventRemoved
	// EventUpdated indicates metadata (such as environment variables) changed.
	EventUpdated
	// EventError communicates a transient enumeration failure.
	EventError
)

// Session describes an interactive desktop session that can host a tray agent.
type Session struct {
	ID         string
	User       string
	UID        uint32
	GID        uint32
	RuntimeDir string
	Display    string
	Env        map[string]string
}

// Equal determines whether two sessions represent the same runtime characteristics.
func (s Session) Equal(other Session) bool {
	if s.ID != other.ID || s.User != other.User || s.UID != other.UID || s.GID != other.GID || s.RuntimeDir != other.RuntimeDir || s.Display != other.Display {
		return false
	}
	if len(s.Env) != len(other.Env) {
		return false
	}
	for k, v := range s.Env {
		if other.Env[k] != v {
			return false
		}
	}
	return true
}

// Event is delivered whenever the session manager observes a change.
type Event struct {
	Type    EventType
	Session Session
	Err     error
}

// Manager tracks interactive user sessions.
type Manager interface {
	List() ([]Session, error)
	Watch(stop <-chan struct{}) (<-chan Event, error)
	Close() error
}

// ErrUnavailable indicates session management is not supported on this platform.
var ErrUnavailable = errors.New("session management unavailable on this platform")
