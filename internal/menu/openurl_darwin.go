//go:build darwin
// +build darwin

package menu

import "os/exec"

func launchURL(raw string) {
	_ = exec.Command("open", raw).Start()
}
