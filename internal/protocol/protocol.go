package protocol

import "github.com/example/gotray/internal/config"

const (
	// CommandMenuGet requests the current menu configuration from the service.
	CommandMenuGet = "menu.get"
)

// Request is the IPC payload sent from user agents to the system service.
type Request struct {
	Token   string `json:"token"`
	Command string `json:"command"`
}

// Response is the IPC reply emitted by the service.
type Response struct {
	Error string            `json:"error,omitempty"`
	Items []config.MenuItem `json:"items,omitempty"`
}
