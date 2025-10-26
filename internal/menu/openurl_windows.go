//go:build windows
// +build windows

package menu

import "os/exec"

func launchURL(raw string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", raw).Start()
}
