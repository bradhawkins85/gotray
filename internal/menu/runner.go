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
	"github.com/example/gotray/internal/logging"
	"github.com/example/gotray/internal/trmm"
)

const defaultRefreshInterval = 30 * time.Second

// Runner handles communication with the system service and synchronises menu
// state for user-session tray processes.
type trayController interface {
	Run(ctx context.Context, updates <-chan UpdatePayload) error
}

// UpdatePayload encapsulates tray menu updates and icon data.
type UpdatePayload struct {
	Items []config.MenuItem
	Icon  []byte
}

type Runner struct {
	secret          string
	refreshInterval time.Duration

	mu             sync.RWMutex
	lastItems      []config.MenuItem
	lastDigest     string
	lastIconDigest string

	tray    trayController
	updates chan UpdatePayload
}

// NewRunner constructs a Runner that loads menu definitions directly from disk
// using the provided secret.
func NewRunner(secret string) *Runner {
	r := &Runner{
		secret:          strings.TrimSpace(secret),
		refreshInterval: defaultRefreshInterval,
	}
	r.tray = newTrayController()
	r.updates = make(chan UpdatePayload, 1)
	return r
}

// Start loads the encrypted configuration from disk and periodically refreshes
// the tray menu. It blocks until the provided context is canceled.
func (r *Runner) Start(ctx context.Context) error {
	if r.secret == "" {
		return errors.New("missing secret; set GOTRAY_SECRET before starting the tray")
	}

	log.Printf("GoTray running in standalone mode")
	logging.Debugf("tray runner initialising with refresh interval %s", r.refreshInterval)

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
	logging.Debugf("performing initial configuration sync")
	if err := r.syncOnce(ctx); err != nil {
		log.Printf("initial sync failed: %v", err)
	}
	if len(r.LatestItems()) == 0 {
		r.publish(nil, nil)
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
			if err := r.syncOnce(ctx); err != nil {
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

func (r *Runner) syncOnce(ctx context.Context) error {
	cfg, err := config.Load(r.secret)
	if err != nil {
		return err
	}
	logging.Debugf("loaded %d menu items from configuration", len(cfg.Items))

	options := trmm.DetectOptions()
	logging.Debugf("detected Tactical RMM options: base=%s agentId=%s site=%d client=%d pk=%d", options.BaseURL, logging.MaskIdentifier(options.AgentID), options.SiteID, options.ClientID, options.AgentPK)
	trayData, err := trmm.FetchTrayData(ctx, nil, options)
	if err != nil {
		log.Printf("Tactical RMM integration failed: %v", err)
	}

	if trayData != nil {
		logging.Debugf("received Tactical RMM tray override with %d items and icon=%t", len(trayData.MenuItems), len(trayData.Icon) > 0)
	} else {
		logging.Debugf("no Tactical RMM tray override available")
	}

	items := make([]config.MenuItem, len(cfg.Items))
	copy(items, cfg.Items)

	seeded := false
	if len(items) == 0 && (trayData == nil || len(trayData.MenuItems) == 0) {
		items = DefaultItems()
		EnsureSequentialOrder(&items)
		cfg.Items = make([]config.MenuItem, len(items))
		copy(cfg.Items, items)
		if err := config.Save(cfg, r.secret); err != nil {
			return err
		}
		seeded = true
		logging.Debugf("seeded configuration with %d default items", len(items))
	} else {
		EnsureSequentialOrder(&items)
		logging.Debugf("retaining %d menu items after local configuration sync", len(items))
	}

	if trayData != nil {
		if len(trayData.MenuItems) > 0 {
			items = make([]config.MenuItem, len(trayData.MenuItems))
			copy(items, trayData.MenuItems)
			EnsureSequentialOrder(&items)
			logging.Debugf("applying %d Tactical RMM menu items", len(items))
		}
	}

	var icon []byte
	if trayData != nil && len(trayData.Icon) > 0 {
		icon = trayData.Icon
		logging.Debugf("using Tactical RMM provided icon (%d bytes)", len(icon))
	}

	r.setTrayState(items, icon)
	if seeded {
		log.Printf("GoTray created a fresh configuration with %d default items", len(items))
	}
	return nil
}

func (r *Runner) setTrayState(items []config.MenuItem, icon []byte) {
	digest := hashItems(items)
	iconDigest := hashBytes(icon)

	r.mu.Lock()
	if digest != "" && digest == r.lastDigest && iconDigest == r.lastIconDigest {
		r.mu.Unlock()
		return
	}
	r.lastItems = make([]config.MenuItem, len(items))
	copy(r.lastItems, items)
	r.lastDigest = digest
	r.lastIconDigest = iconDigest
	r.mu.Unlock()
	logging.Debugf("published tray state with %d items (digest=%s iconDigest=%s)", len(items), digest, iconDigest)
	r.publish(items, icon)
}

func (r *Runner) publish(items []config.MenuItem, icon []byte) {
	if r.updates == nil {
		return
	}

	if len(items) == 0 {
		items = DefaultItems()
	}

	payload := make([]config.MenuItem, len(items))
	copy(payload, items)

	update := UpdatePayload{
		Items: payload,
		Icon:  cloneIcon(icon),
	}

	select {
	case r.updates <- update:
	default:
		select {
		case <-r.updates:
		default:
		}
		select {
		case r.updates <- update:
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

func hashBytes(icon []byte) string {
	normalized := normalizedIcon(icon)
	if len(normalized) == 0 {
		return ""
	}
	sum := sha256.Sum256(normalized)
	return hex.EncodeToString(sum[:])
}
