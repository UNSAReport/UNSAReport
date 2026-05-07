//go:build !windows

package capture

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/creack/pty"
)

func RunPTY(ctx context.Context, cwd string, command string, inputs []TimedInput) (string, error) {
	shell := "bash"
	shellArgs := []string{"-c", command}

	cmd := exec.CommandContext(ctx, shell, shellArgs...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1")

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: 100, Rows: 30})
	if err != nil {
		return "", err
	}
	defer ptmx.Close()

	var buf bytes.Buffer
	readDone := make(chan error, 1)
	go func() {
		_, err := io.Copy(&buf, ptmx)
		readDone <- err
	}()

	// Replay inputs.
	for _, in := range inputs {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			return "", ctx.Err()
		case <-time.After(in.Delay):
			_, _ = ptmx.Write([]byte(in.Text + "\r"))
		}
	}

	if len(inputs) > 0 {
		time.Sleep(200 * time.Millisecond)
		// Ctrl-D
		_, _ = ptmx.Write([]byte{0x04})
	}

	waitErr := cmd.Wait()
	_ = ptmx.Close()
	_ = <-readDone

	return buf.String(), waitErr
}
