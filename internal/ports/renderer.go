package ports

import (
	"context"
	"time"
)

// CaptureCommand represents a single terminal command to be executed during rendering.
type CaptureCommand struct {
	Type  string
	Args  string
	Delay time.Duration
}

// Renderer abstracts the terminal capture engine that executes commands and produces rendered output.
type Renderer interface {
	Render(ctx context.Context, resultPath string, commands []CaptureCommand, flags []string, cfg CaptureConfig) (string, error)
}
