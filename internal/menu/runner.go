package menu

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/example/gotray/internal/config"
)

const defaultRefreshInterval = 30 * time.Second

// Runner handles communication with the system service and synchronises menu
// state for user-session tray processes.
type trayController interface {
	Run(ctx context.Context, updates <-chan []config.MenuItem) error
}

type Runner struct {
	secret          string
	refreshInterval time.Duration

	mu         sync.RWMutex
	lastItems  []config.MenuItem
	lastDigest string

	tray    trayController
	updates chan []config.MenuItem
}

// NewRunner constructs a Runner that loads menu definitions directly from disk
// using the provided secret.
func NewRunner(secret string) *Runner {
	r := &Runner{
		secret:          strings.TrimSpace(secret),
		refreshInterval: defaultRefreshInterval,
	}
	r.tray = newTrayController()
	r.updates = make(chan []config.MenuItem, 1)
	return r
}

// Start loads the encrypted configuration from disk and periodically refreshes
// the tray menu. It blocks until the provided context is canceled.
func (r *Runner) Start(ctx context.Context) error {
	if r.secret == "" {
		return errors.New("missing secret; set GOTRAY_SECRET before starting the tray")
	}

	log.Printf("GoTray running in standalone mode")

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
	if err := r.syncOnce(); err != nil {
		log.Printf("initial sync failed: %v", err)
	}
	if len(r.LatestItems()) == 0 {
		r.publish(nil)
	} else {
		log.Printf("GoTray loaded %d menu items", len(r.LatestItems()))
	}

	ticker := time.NewTicker(r.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("GoTray tray agent stopping")
			return ctx.Err()
		case <-ticker.C:
			if err := r.syncOnce(); err != nil {
				log.Printf("tray refresh failed: %v", err)
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

func (r *Runner) syncOnce() error {
	cfg, err := config.Load(r.secret)
	if err != nil {
		return err
	}

	seeded := false
	if len(cfg.Items) == 0 {
		cfg.Items = DefaultItems()
		EnsureSequentialOrder(&cfg.Items)
		if err := config.Save(cfg, r.secret); err != nil {
			return err
		}
		seeded = true
	} else {
		EnsureSequentialOrder(&cfg.Items)
	}

	r.setItems(cfg.Items)
	if seeded {
		log.Printf("GoTray created a fresh configuration with %d default items", len(cfg.Items))
	}
	return nil
}

func (r *Runner) setItems(items []config.MenuItem) {
	digest := hashItems(items)

	r.mu.Lock()
	if digest != "" && digest == r.lastDigest {
		r.mu.Unlock()
		return
	}
	r.lastItems = make([]config.MenuItem, len(items))
	copy(r.lastItems, items)
	r.lastDigest = digest
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

func hashItems(items []config.MenuItem) string {
	if len(items) == 0 {
		return ""
	}

	payload, err := json.Marshal(items)
	if err != nil {
		return ""
	}

	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
