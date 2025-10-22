//go:build windows

package service

import (
	"os/exec"

	"github.com/example/gotray/internal/service/sessions"
)

func applySessionCredentials(cmd *exec.Cmd, sess sessions.Session) {
	// Windows services require impersonation to drop privileges. Stubbed for now.
}

func terminateProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
