package capture

import (
	"bytes"
	"context"
	"io"
	"os"
	"time"

	"github.com/aymanbagabas/go-pty"
)

func RunPTY(ctx context.Context, cwd string, command string, inputs []TimedInput) (string, error) {
	p, err := pty.New()
	if err != nil {
		return "", err
	}
	defer p.Close()

	if err := p.Resize(100, 30); err != nil {
		return "", err
	}

	shell := "bash"
	shellArgs := []string{"-c", command}

	cmd := p.CommandContext(ctx, shell, shellArgs...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1")

	if err := cmd.Start(); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	readDone := make(chan error, 1)
	go func() {
		_, err := io.Copy(&buf, p)
		readDone <- err
	}()

	// Replay inputs.
	for _, in := range inputs {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			return "", ctx.Err()
		case <-time.After(in.Delay):
			_, _ = p.Write([]byte(in.Text + "\r"))
		}
	}

	if len(inputs) > 0 {
		time.Sleep(200 * time.Millisecond)
		// Ctrl-D
		_, _ = p.Write([]byte{0x04})
	}

	waitErr := cmd.Wait()
	_ = p.Close()
	_ = <-readDone

	return buf.String(), waitErr
}
