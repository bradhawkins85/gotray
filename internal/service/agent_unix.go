//go:build unix

package service

import (
	"os/exec"
	"syscall"

	"github.com/example/gotray/internal/service/sessions"
)

func applySessionCredentials(cmd *exec.Cmd, sess sessions.Session) {
	if sess.UID == 0 {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: sess.UID,
			Gid: sess.GID,
		},
	}
}

func terminateProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Signal(syscall.SIGTERM)
}
