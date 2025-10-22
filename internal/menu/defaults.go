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
