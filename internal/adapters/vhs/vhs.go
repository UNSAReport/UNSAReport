package vhs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Render(ctx context.Context, tapePath string) error {
	cmd := exec.CommandContext(ctx, "vhs", tapePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vhs command failed: %w", err)
	}
	return nil
}
