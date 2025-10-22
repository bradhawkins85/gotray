package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/example/gotray/internal/service/sessions"
)

type execAgent struct {
	cmd       *exec.Cmd
	tokenFile string
	done      chan struct{}
	mu        sync.Mutex
	err       error
}

func launchAgentProcess(ctx context.Context, sess sessions.Session, token, endpoint string) (agentProcess, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable: %w", err)
	}

	tokenFile, err := writeTokenFile(sess, token)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, exe, "tray")
	cmd.Env = buildAgentEnv(sess, endpoint, tokenFile)
	cmd.Dir = sess.RuntimeDir
	applySessionCredentials(cmd, sess)

	agent := &execAgent{
		cmd:       cmd,
		tokenFile: tokenFile,
		done:      make(chan struct{}),
	}

	if err := cmd.Start(); err != nil {
		os.Remove(tokenFile)
		return nil, fmt.Errorf("start tray: %w", err)
	}

	go agent.wait()
	return agent, nil
}

func (a *execAgent) wait() {
	err := a.cmd.Wait()
	if a.tokenFile != "" {
		_ = os.Remove(a.tokenFile)
	}
	a.mu.Lock()
	a.err = err
	a.mu.Unlock()
	close(a.done)
}

func (a *execAgent) Done() <-chan struct{} {
	return a.done
}

func (a *execAgent) Err() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.err
}

func (a *execAgent) Stop() error {
	if a.cmd.Process == nil {
		return nil
	}
	if err := terminateProcess(a.cmd); err != nil {
		if !errors.Is(err, os.ErrProcessDone) {
			return err
		}
	}
	select {
	case <-a.done:
		return nil
	case <-time.After(5 * time.Second):
		return a.cmd.Process.Kill()
	}
}

func buildAgentEnv(sess sessions.Session, endpoint, tokenFile string) []string {
	env := os.Environ()
	for k, v := range sess.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	env = append(env, fmt.Sprintf("GOTRAY_SERVICE_ENDPOINT=%s", endpoint))
	env = append(env, fmt.Sprintf("GOTRAY_SERVICE_TOKEN_FILE=%s", tokenFile))
	if sess.RuntimeDir != "" {
		env = append(env, fmt.Sprintf("XDG_RUNTIME_DIR=%s", sess.RuntimeDir))
	}
	if sess.Display != "" {
		env = append(env, fmt.Sprintf("DISPLAY=%s", sess.Display))
	}
	return env
}

func writeTokenFile(sess sessions.Session, token string) (string, error) {
	if token == "" {
		return "", errors.New("missing service token")
	}

	dir := sess.RuntimeDir
	if dir == "" {
		dir = os.TempDir()
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("ensure runtime dir: %w", err)
	}
	path := filepath.Join(dir, fmt.Sprintf("gotray-token-%d", time.Now().UnixNano()))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", fmt.Errorf("create token file: %w", err)
	}
	if _, err := file.WriteString(token); err != nil {
		file.Close()
		os.Remove(path)
		return "", fmt.Errorf("write token file: %w", err)
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("close token file: %w", err)
	}
	if sess.UID != 0 {
		_ = os.Chown(path, int(sess.UID), int(sess.GID))
	}
	return path, nil
}
