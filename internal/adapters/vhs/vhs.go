package vhs

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/christianmz565/lab-report/internal/dependencies"
)

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Render(ctx context.Context, tapePath string) error {
	if err := dependencies.Check(dependencies.VHS); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "vhs", tapePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vhs command failed: %w", err)
	}
	return nil
}
