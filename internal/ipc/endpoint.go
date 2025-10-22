package ipc

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"
)

const defaultServicePort = "127.0.0.1:47863"

// Endpoint describes how the system service communicates with user-session agents.
type Endpoint struct {
	Network string
	Address string
}

// DefaultEndpoint resolves the listening endpoint using environment overrides.
func DefaultEndpoint() Endpoint {
	if addr := strings.TrimSpace(os.Getenv("GOTRAY_SERVICE_ADDR")); addr != "" {
		return Endpoint{Network: "tcp", Address: addr}
	}

	if runtime.GOOS == "windows" {
		return Endpoint{Network: "tcp", Address: defaultServicePort}
	}

	return Endpoint{Network: "tcp", Address: defaultServicePort}
}

// Listen binds to the configured endpoint.
func (e Endpoint) Listen() (net.Listener, error) {
	return net.Listen(e.Network, e.Address)
}

// DialContext establishes a client connection with sensible timeouts.
func (e Endpoint) DialContext(ctx context.Context) (net.Conn, error) {
	d := &net.Dialer{Timeout: 5 * time.Second}
	return d.DialContext(ctx, e.Network, e.Address)
}

// String provides a readable representation for logs.
func (e Endpoint) String() string {
	return fmt.Sprintf("%s://%s", e.Network, e.Address)
}
