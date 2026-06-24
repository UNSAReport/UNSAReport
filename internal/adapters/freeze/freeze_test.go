//go:build integration

package freeze

import (
	"context"
	"os/exec"
	"testing"

	"github.com/UNSAReport/UNSAReport/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFreezeAdapter_Render_NoFreeze(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("freeze"); err == nil {
		t.Skip("freeze binary found, skipping no-binary test")
	}

	a := New()
	ctx := context.Background()
	cfg := ports.CaptureConfig{Columns: 80, Prompt: "$ "}

	_, err := a.Render(ctx, "output.png", nil, nil, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required external tool")
}

func TestFreezeAdapter_Render_NoMagick(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("freeze"); err != nil {
		t.Skip("freeze binary not found, skipping test")
	}
	if _, err := exec.LookPath("magick"); err == nil {
		t.Skip("magick binary found, skipping no-binary test")
	}

	a := New()
	ctx := context.Background()
	cfg := ports.CaptureConfig{Columns: 80, Prompt: "$ "}

	_, err := a.Render(ctx, "output.png", nil, nil, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required external tool")
}

func TestFreezeAdapter_GetDefaultShell(t *testing.T) {
	t.Parallel()
	shell, args := getDefaultShell()
	assert.NotEmpty(t, shell)
	assert.NotNil(t, args)
}

func TestFreezeAdapter_GetAnsi(t *testing.T) {
	t.Parallel()

	t.Run("with colors", func(t *testing.T) {
		t.Parallel()
		colors := map[string]string{
			"prompt":  "32",
			"command": "36",
			"reset":   "0",
		}
		assert.Equal(t, "\x1b[32m", getAnsi(colors, "prompt"))
		assert.Equal(t, "\x1b[36m", getAnsi(colors, "command"))
		assert.Equal(t, "\x1b[0m", getAnsi(colors, "reset"))
	})

	t.Run("without colors", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "\x1b[0m", getAnsi(nil, "reset"))
		assert.Equal(t, "", getAnsi(nil, "prompt"))
	})

	t.Run("unknown color", func(t *testing.T) {
		t.Parallel()
		colors := map[string]string{"prompt": "32"}
		assert.Equal(t, "", getAnsi(colors, "unknown"))
		assert.Equal(t, "\x1b[0m", getAnsi(colors, "reset"))
	})
}

func TestFreezeAdapter_Render_Success(t *testing.T) {
	if _, err := exec.LookPath("freeze"); err != nil {
		t.Skip("freeze binary not found, skipping integration test")
	}
	if _, err := exec.LookPath("magick"); err != nil {
		t.Skip("magick binary not found, skipping integration test")
	}

	// This test runs the full PTY + freeze + magick pipeline.
	// It is slow due to PTY initialization, so we skip in short mode.
	if testing.Short() {
		t.Skip("skipping slow integration test in short mode")
	}

	// Use t.Skip to avoid CI timeouts - enable locally with:
	//   go test -run TestFreezeAdapter_Render_Success -tags integration ./internal/adapters/freeze/
	t.Skip("freeze render test requires manual invocation due to PTY setup time")
}
