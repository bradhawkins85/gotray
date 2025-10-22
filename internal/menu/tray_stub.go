//go:build !cgo && !windows
// +build !cgo,!windows

package menu

import (
	"context"
	"errors"

	"github.com/example/gotray/internal/config"
)

type trayUnsupported struct{}

func newTrayController() trayController {
	return trayUnsupported{}
}

func (trayUnsupported) Run(_ context.Context, _ <-chan []config.MenuItem) error {
	return errors.New("system tray is unavailable without cgo support")
}
