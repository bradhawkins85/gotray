package menu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/example/gotray/internal/config"
	"github.com/example/gotray/internal/ipc"
	"github.com/example/gotray/internal/protocol"
	"github.com/example/gotray/internal/security"
)

const defaultRefreshInterval = 30 * time.Second

// Runner handles communication with the system service and synchronises menu
// state for user-session tray processes.
type Runner struct {
	token           string
	endpoint        ipc.Endpoint
	refreshInterval time.Duration

	mu        sync.RWMutex
	lastItems []config.MenuItem
}

// NewRunner constructs a Runner that authenticates with the configured service.
func NewRunner(secret string) *Runner {
	return &Runner{
		token:           security.ResolveServiceToken(secret),
		endpoint:        ipc.DefaultEndpoint(),
		refreshInterval: defaultRefreshInterval,
	}
}

// Start connects to the system service and periodically refreshes the menu
// definition. It blocks until the provided context is canceled.
func (r *Runner) Start(ctx context.Context) error {
	if r.token == "" {
		return errors.New("missing service token; set GOTRAY_SERVICE_TOKEN or GOTRAY_SECRET")
	}

	log.Printf("GoTray tray agent connecting to %s", r.endpoint.String())

	// Perform an initial sync before entering the refresh loop.
	if err := r.syncOnce(ctx); err != nil {
		log.Printf("initial sync failed: %v", err)
	}

	ticker := time.NewTicker(r.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("GoTray tray agent stopping")
			return ctx.Err()
		case <-ticker.C:
			if err := r.syncOnce(ctx); err != nil {
				log.Printf("tray sync failed: %v", err)
			}
		}
	}
}

// LatestItems returns the most recently downloaded menu entries.
func (r *Runner) LatestItems() []config.MenuItem {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]config.MenuItem, len(r.lastItems))
	copy(out, r.lastItems)
	return out
}

func (r *Runner) syncOnce(ctx context.Context) error {
	conn, err := r.endpoint.DialContext(ctx)
	if err != nil {
		return fmt.Errorf("connect to service: %w", err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))

	req := protocol.Request{Token: r.token, Command: protocol.CommandMenuGet}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return fmt.Errorf("send request: %w", err)
	}

	var resp protocol.Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.Error != "" {
		return errors.New(resp.Error)
	}

	r.mu.Lock()
	r.lastItems = make([]config.MenuItem, len(resp.Items))
	copy(r.lastItems, resp.Items)
	r.mu.Unlock()

	log.Printf("GoTray tray agent synchronized %d menu items", len(resp.Items))
	return nil
}
