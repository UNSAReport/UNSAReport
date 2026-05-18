package ports

import (
	"context"
	"time"
)

type CaptureCommand struct {
	Type  string
	Args  string
	Delay time.Duration
}

type Renderer interface {
	Render(ctx context.Context, resultPath string, commands []CaptureCommand, flags []string, cfg CaptureConfig) (string, error)
}
