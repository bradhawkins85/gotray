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
type trayController interface {
	Run(ctx context.Context, updates <-chan []config.MenuItem) error
}

type Runner struct {
	token           string
	endpoint        ipc.Endpoint
	refreshInterval time.Duration

	mu        sync.RWMutex
	lastItems []config.MenuItem

	tray    trayController
	updates chan []config.MenuItem
}

// NewRunner constructs a Runner that authenticates with the configured service.
func NewRunner(secret string) *Runner {
	r := &Runner{
		token:           security.ResolveServiceToken(secret),
		endpoint:        ipc.DefaultEndpoint(),
		refreshInterval: defaultRefreshInterval,
	}
	r.tray = newTrayController()
	r.updates = make(chan []config.MenuItem, 1)
	return r
}

// Start connects to the system service and periodically refreshes the menu
// definition. It blocks until the provided context is canceled.
func (r *Runner) Start(ctx context.Context) error {
	if r.token == "" {
		return errors.New("missing service token; set GOTRAY_SERVICE_TOKEN or GOTRAY_SECRET")
	}

	log.Printf("GoTray tray agent connecting to %s", r.endpoint.String())

	var trayErr <-chan error
	if r.tray != nil {
		ch := make(chan error, 1)
		trayErr = ch
		go func() {
			ch <- r.tray.Run(ctx, r.updates)
		}()
	}
	defer func() {
		if r.updates != nil {
			close(r.updates)
		}
	}()

	// Perform an initial sync before entering the refresh loop.
	if err := r.syncOnce(ctx); err != nil {
		log.Printf("initial sync failed: %v", err)
	}
	if len(r.LatestItems()) == 0 {
		r.publish(nil)
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
		case err := <-trayErr:
			return err
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

	r.setItems(resp.Items)
	log.Printf("GoTray tray agent synchronized %d menu items", len(resp.Items))
	return nil
}

func (r *Runner) setItems(items []config.MenuItem) {
	r.mu.Lock()
	r.lastItems = make([]config.MenuItem, len(items))
	copy(r.lastItems, items)
	r.mu.Unlock()
	r.publish(items)
}

func (r *Runner) publish(items []config.MenuItem) {
	if r.updates == nil {
		return
	}

	if len(items) == 0 {
		items = DefaultItems()
	}

	payload := make([]config.MenuItem, len(items))
	copy(payload, items)

	select {
	case r.updates <- payload:
	default:
		select {
		case <-r.updates:
		default:
		}
		select {
		case r.updates <- payload:
		default:
		}
	}
}
