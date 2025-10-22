//go:build cgo || windows
// +build cgo windows

package menu

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"sort"
	"sync"

	"github.com/getlantern/systray"

	"github.com/example/gotray/internal/config"
)

type systrayController struct {
	mu      sync.Mutex
	entries []trayEntry
}

type trayEntry struct {
	item   *systray.MenuItem
	cancel context.CancelFunc
}

func newTrayController() trayController {
	return &systrayController{}
}

func (c *systrayController) Run(ctx context.Context, updates <-chan []config.MenuItem) error {
	done := make(chan struct{})

	go systray.Run(func() {
		if iconData != nil {
			systray.SetIcon(iconData)
			if runtime.GOOS == "darwin" {
				systray.SetTemplateIcon(iconData, iconData)
			}
		}
		systray.SetTooltip("GoTray")

		quit := systray.AddMenuItem("Quit GoTray", "Exit the application")
		go func() {
			for {
				select {
				case <-ctx.Done():
					systray.Quit()
					return
				case <-quit.ClickedCh:
					systray.Quit()
					return
				}
			}
		}()

		go c.listen(ctx, updates)
	}, func() {
		c.shutdown()
		close(done)
	})

	select {
	case <-ctx.Done():
		systray.Quit()
		<-done
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (c *systrayController) listen(ctx context.Context, updates <-chan []config.MenuItem) {
	for {
		select {
		case <-ctx.Done():
			systray.Quit()
			return
		case items, ok := <-updates:
			if !ok {
				systray.Quit()
				return
			}
			c.render(ctx, items)
		}
	}
}

func (c *systrayController) render(ctx context.Context, items []config.MenuItem) {
	if len(items) == 0 {
		items = DefaultItems()
	}

	c.mu.Lock()
	old := c.entries
	c.entries = nil
	c.mu.Unlock()

	for _, entry := range old {
		entry.cancel()
		if entry.item != nil {
			entry.item.Hide()
		}
	}

	grouped := groupByParent(items)
	newEntries := c.renderGroup(ctx, grouped, "", nil)

	c.mu.Lock()
	c.entries = newEntries
	c.mu.Unlock()
}

func (c *systrayController) renderGroup(ctx context.Context, grouped map[string][]config.MenuItem, parentID string, parent *systray.MenuItem) []trayEntry {
	entries := make([]trayEntry, 0)
	for _, item := range grouped[parentID] {
		entries = append(entries, c.addMenuItem(ctx, item, parent, grouped)...)
	}
	return entries
}

func (c *systrayController) addMenuItem(ctx context.Context, item config.MenuItem, parent *systray.MenuItem, grouped map[string][]config.MenuItem) []trayEntry {
	switch item.Type {
	case config.MenuItemDivider:
		if parent == nil {
			systray.AddSeparator()
			return nil
		}
		mi := parent.AddSubMenuItem("—", item.Description)
		mi.Disable()
		return []trayEntry{{item: mi, cancel: func() {}}}
	case config.MenuItemMenu:
		mi := c.makeMenuItem(parent, item)
		ctxItem, cancel := context.WithCancel(ctx)
		go drainClicks(ctxItem, mi.ClickedCh)
		entries := []trayEntry{{item: mi, cancel: cancel}}
		entries = append(entries, c.renderGroup(ctx, grouped, item.ID, mi)...)
		return entries
	case config.MenuItemText:
		mi := c.makeMenuItem(parent, item)
		ctxItem, cancel := context.WithCancel(ctx)
		go drainClicks(ctxItem, mi.ClickedCh)
		return []trayEntry{{item: mi, cancel: cancel}}
	case config.MenuItemCommand:
		mi := c.makeMenuItem(parent, item)
		ctxItem, cancel := context.WithCancel(ctx)
		go func(ch <-chan struct{}, cmd string, args []string, dir string) {
			for {
				select {
				case <-ctxItem.Done():
					return
				case _, ok := <-ch:
					if !ok {
						return
					}
					go executeCommand(ctx, cmd, args, dir)
				}
			}
		}(mi.ClickedCh, item.Command, item.Arguments, item.WorkingDir)
		return []trayEntry{{item: mi, cancel: cancel}}
	case config.MenuItemURL:
		mi := c.makeMenuItem(parent, item)
		ctxItem, cancel := context.WithCancel(ctx)
		go func(ch <-chan struct{}, target string) {
			for {
				select {
				case <-ctxItem.Done():
					return
				case _, ok := <-ch:
					if !ok {
						return
					}
					go openURL(target)
				}
			}
		}(mi.ClickedCh, item.URL)
		return []trayEntry{{item: mi, cancel: cancel}}
	default:
		mi := c.makeMenuItem(parent, config.MenuItem{
			Label:       fmt.Sprintf("Unsupported: %s", item.Type),
			Description: item.Description,
		})
		mi.Disable()
		return []trayEntry{{item: mi, cancel: func() {}}}
	}
}

func (c *systrayController) makeMenuItem(parent *systray.MenuItem, item config.MenuItem) *systray.MenuItem {
	if parent == nil {
		return systray.AddMenuItem(item.Label, item.Description)
	}
	return parent.AddSubMenuItem(item.Label, item.Description)
}

func drainClicks(ctx context.Context, ch <-chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ch:
			if !ok {
				return
			}
		}
	}
}

func groupByParent(items []config.MenuItem) map[string][]config.MenuItem {
	grouped := make(map[string][]config.MenuItem)
	for _, item := range items {
		key := item.ParentID
		grouped[key] = append(grouped[key], item)
	}
	for key := range grouped {
		sort.SliceStable(grouped[key], func(i, j int) bool {
			if grouped[key][i].Order == grouped[key][j].Order {
				return grouped[key][i].ID < grouped[key][j].ID
			}
			return grouped[key][i].Order < grouped[key][j].Order
		})
	}
	return grouped
}

func (c *systrayController) shutdown() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, entry := range c.entries {
		entry.cancel()
	}
	c.entries = nil
}

func executeCommand(ctx context.Context, command string, args []string, workingDir string) {
	if command == "" {
		return
	}

	cmd := exec.CommandContext(ctx, command, args...)
	if workingDir != "" {
		cmd.Dir = workingDir
	}
	_ = cmd.Start()
}

func openURL(raw string) {
	if raw == "" {
		return
	}
	if _, err := url.ParseRequestURI(raw); err != nil {
		return
	}

	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", raw).Start()
	case "darwin":
		_ = exec.Command("open", raw).Start()
	default:
		_ = exec.Command("xdg-open", raw).Start()
	}
}
