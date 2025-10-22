package menu

import (
	"time"

	"github.com/example/gotray/internal/config"
)

// DefaultItems returns baseline menu options when no configuration exists.
func DefaultItems() []config.MenuItem {
	now := time.Now().UTC().Format(time.RFC3339)
	return []config.MenuItem{
		{
			ID:          "10",
			Order:       10,
			Type:        config.MenuItemText,
			Label:       "GoTray",
			Description: "GoTray is running",
			CreatedUTC:  now,
			UpdatedUTC:  now,
		},
		{
			ID:          "20",
			Order:       20,
			Type:        config.MenuItemURL,
			Label:       "Visit Project",
			URL:         "https://example.com",
			Description: "Open the GoTray project page",
			CreatedUTC:  now,
			UpdatedUTC:  now,
		},
		{
			ID:          "30",
			Order:       30,
			Type:        config.MenuItemQuit,
			Label:       "Quit GoTray",
			Description: "Exit the GoTray application",
			CreatedUTC:  now,
			UpdatedUTC:  now,
		},
	}
}
