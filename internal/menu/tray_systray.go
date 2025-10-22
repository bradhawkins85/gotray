//go:build cgo || windows
// +build cgo windows

package menu

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
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

	newEntries := make([]trayEntry, 0, len(items))
	for _, item := range items {
		if entry := c.addMenuItem(ctx, item); entry != nil {
			newEntries = append(newEntries, *entry)
		}
	}

	c.mu.Lock()
	c.entries = newEntries
	c.mu.Unlock()
}

func (c *systrayController) addMenuItem(ctx context.Context, item config.MenuItem) *trayEntry {
	switch item.Type {
	case config.MenuItemDivider:
		systray.AddSeparator()
		return nil
	case config.MenuItemText:
		mi := systray.AddMenuItem(item.Label, item.Description)
		ctxItem, cancel := context.WithCancel(ctx)
		go func(ch <-chan struct{}) {
			for {
				select {
				case <-ctxItem.Done():
					return
				case _, ok := <-ch:
					if !ok {
						return
					}
				}
			}
		}(mi.ClickedCh)
		return &trayEntry{item: mi, cancel: cancel}
	case config.MenuItemCommand:
		mi := systray.AddMenuItem(item.Label, item.Description)
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
		return &trayEntry{item: mi, cancel: cancel}
	case config.MenuItemURL:
		mi := systray.AddMenuItem(item.Label, item.Description)
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
		return &trayEntry{item: mi, cancel: cancel}
	default:
		mi := systray.AddMenuItem(fmt.Sprintf("Unsupported: %s", item.Type), item.Description)
		mi.Disable()
		return &trayEntry{item: mi, cancel: func() {}}
	}
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
