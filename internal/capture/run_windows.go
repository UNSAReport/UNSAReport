//go:build windows

package capture

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"time"
)

// Windows fallback: no PTY. Best-effort for non-interactive commands.
func RunPTY(ctx context.Context, cwd string, command string, inputs []TimedInput) (string, error) {
	cmd := exec.CommandContext(ctx, "cmd.exe", "/c", command)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := cmd.Start(); err != nil {
		return "", err
	}

	go io.Copy(&buf, stdout)
	go io.Copy(&buf, stderr)

	for _, in := range inputs {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			return "", ctx.Err()
		case <-time.After(in.Delay):
			_, _ = stdin.Write([]byte(in.Text + "\r\n"))
		}
	}
	_ = stdin.Close()

	err = cmd.Wait()
	return buf.String(), err
}
