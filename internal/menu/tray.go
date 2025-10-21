package menu

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	"github.com/getlantern/systray"

	"github.com/example/gotray/internal/config"
)

// Runner handles creation of systray menu items.
type Runner struct {
	config *config.Config
}

// NewRunner constructs a Runner with the given configuration.
func NewRunner(cfg *config.Config) *Runner {
	return &Runner{config: cfg}
}

// Start initializes the tray lifecycle.
func (r *Runner) Start(ctx context.Context) error {
	if r.config == nil {
		return errors.New("nil configuration")
	}

	systray.Run(func() {
		if iconData != nil {
			systray.SetIcon(iconData)
			if runtime.GOOS == "darwin" {
				systray.SetTemplateIcon(iconData, iconData)
			}
		}
		systray.SetTooltip("GoTray")

		for _, item := range r.config.Items {
			r.addMenuItem(ctx, item)
		}

		quit := systray.AddMenuItem("Quit GoTray", "Exit the application")
		go func() {
			<-quit.ClickedCh
			systray.Quit()
		}()
	}, nil)
	return nil
}

func (r *Runner) addMenuItem(ctx context.Context, item config.MenuItem) {
	switch item.Type {
	case config.MenuItemDivider:
		systray.AddSeparator()
		return
	case config.MenuItemText:
		mi := systray.AddMenuItem(item.Label, item.Description)
		go func() {
			for range mi.ClickedCh {
				// Text items have no action.
			}
		}()
	case config.MenuItemCommand:
		mi := systray.AddMenuItem(item.Label, item.Description)
		go func(cmd string, args []string, dir string) {
			for range mi.ClickedCh {
				go executeCommand(ctx, cmd, args, dir)
			}
		}(item.Command, item.Arguments, item.WorkingDir)
	case config.MenuItemURL:
		mi := systray.AddMenuItem(item.Label, item.Description)
		go func(target string) {
			for range mi.ClickedCh {
				go openURL(target)
			}
		}(item.URL)
	default:
		mi := systray.AddMenuItem(fmt.Sprintf("Unsupported: %s", item.Type), item.Description)
		mi.Disable()
	}
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

// DefaultItems returns baseline menu options when no configuration exists.
func DefaultItems() []config.MenuItem {
	now := time.Now().UTC().Format(time.RFC3339)
	return []config.MenuItem{
		{
			ID:          "welcome",
			Type:        config.MenuItemText,
			Label:       "GoTray",
			Description: "GoTray is running",
			CreatedUTC:  now,
			UpdatedUTC:  now,
		},
		{
			ID:          "docs",
			Type:        config.MenuItemURL,
			Label:       "Visit Project",
			URL:         "https://example.com",
			Description: "Open the GoTray project page",
			CreatedUTC:  now,
			UpdatedUTC:  now,
		},
		{
			ID:         "divider",
			Type:       config.MenuItemDivider,
			CreatedUTC: now,
			UpdatedUTC: now,
		},
	}
}
