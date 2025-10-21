package menu

import "github.com/example/gotray/internal/config"

// Runner handles creation of systray menu items.
type Runner struct {
	config *config.Config
}

// NewRunner constructs a Runner with the given configuration.
func NewRunner(cfg *config.Config) *Runner {
	return &Runner{config: cfg}
}
