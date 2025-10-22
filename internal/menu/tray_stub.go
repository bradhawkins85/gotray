//go:build !cgo && !windows
// +build !cgo,!windows

package menu

import (
	"context"
	"errors"
)

// Start returns an error indicating tray functionality is unavailable without cgo.
func (r *Runner) Start(_ context.Context) error {
	return errors.New("system tray is unavailable without cgo support")
}
