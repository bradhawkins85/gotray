//go:build !windows && !darwin
// +build !windows,!darwin

package menu

import "os/exec"

func launchURL(raw string) {
	_ = exec.Command("xdg-open", raw).Start()
}
