//go:build !cgo && !windows
// +build !cgo,!windows

package menu

import (
	"context"
	"errors"
)

type trayUnsupported struct{}

func newTrayController(_ func()) trayController {
	return trayUnsupported{}
}

func (trayUnsupported) Run(_ context.Context, _ <-chan UpdatePayload) error {
	return errors.New("system tray is unavailable without cgo support")
}
