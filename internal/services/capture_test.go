package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/UNSAReport/UNSAReport/internal/mocks"
	"github.com/UNSAReport/UNSAReport/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCaptureService_Execute_NoArgs(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/tmp/test", nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", mock.Anything).
		Return("/tmp/test", ports.UnsareportConfig{}, false, nil)
	cfg.On("ReadConfig", mock.Anything).
		Return(ports.UnsareportConfig{
			Capture: ports.CaptureConfig{Columns: 120, Prompt: "❯ ", Colors: map[string]string{"prompt": "32"}},
		}, true, nil)

	var stdout bytes.Buffer
	svc := NewCaptureService(
		WithCaptureRenderer(mocks.NewRenderer(t)),
		WithCaptureFS(fs),
		WithCaptureConfig(cfg),
		WithCaptureStdout(&stdout),
		WithCaptureStderr(io.Discard),
	)

	err := svc.Execute(context.Background(), CaptureOptions{
		Args: []string{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "result image path is required")
}

func TestCaptureService_Execute_RendererError(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/tmp/test", nil)
	fs.On("EnsureDir", mock.Anything).Return(nil).Maybe()
	fs.On("WriteFileAtomic", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", mock.Anything).
		Return("/tmp/test", ports.UnsareportConfig{}, false, nil)
	cfg.On("ReadConfig", mock.Anything).
		Return(ports.UnsareportConfig{
			Capture: ports.CaptureConfig{Columns: 120, Prompt: "❯ ", Colors: map[string]string{"prompt": "32"}},
		}, true, nil)

	renderer := mocks.NewRenderer(t)
	renderer.On("Render", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("", fmt.Errorf("freeze not found"))

	var stdout bytes.Buffer
	svc := NewCaptureService(
		WithCaptureRenderer(renderer),
		WithCaptureFS(fs),
		WithCaptureConfig(cfg),
		WithCaptureStdout(&stdout),
		WithCaptureStderr(io.Discard),
	)

	err := svc.Execute(context.Background(), CaptureOptions{
		Args: []string{"output.png", "ls"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render")
}

func TestCaptureService_Execute_InstructionParsing(t *testing.T) {
	t.Parallel()

	var capturedCommands []ports.CaptureCommand
	renderer := mocks.NewRenderer(t)
	renderer.On("Render", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			capturedCommands = args.Get(2).([]ports.CaptureCommand)
		}).
		Return("", nil)

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/tmp/test", nil)
	fs.On("EnsureDir", mock.Anything).Return(nil).Maybe()
	fs.On("WriteFileAtomic", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", mock.Anything).
		Return("/tmp/test", ports.UnsareportConfig{}, false, nil)
	cfg.On("ReadConfig", mock.Anything).
		Return(ports.UnsareportConfig{
			Capture: ports.CaptureConfig{Columns: 120, Prompt: "❯ ", Colors: map[string]string{"prompt": "32"}},
		}, true, nil)

	var stdout bytes.Buffer
	svc := NewCaptureService(
		WithCaptureRenderer(renderer),
		WithCaptureFS(fs),
		WithCaptureConfig(cfg),
		WithCaptureStdout(&stdout),
		WithCaptureStderr(io.Discard),
	)

	// Test wait instruction
	err := svc.Execute(context.Background(), CaptureOptions{
		Args: []string{"output.png", "w:500ms", "ls -la"},
	})
	require.NoError(t, err)

	var foundWait bool
	for _, cmd := range capturedCommands {
		if cmd.Type == "Sleep" && cmd.Delay == 500*time.Millisecond {
			foundWait = true
			break
		}
	}
	assert.True(t, foundWait, "should have parsed wait instruction")

	// Test raw instruction
	err = svc.Execute(context.Background(), CaptureOptions{
		Args: []string{"output.png", "r:echo hello"},
	})
	require.NoError(t, err)

	var foundRaw bool
	for _, cmd := range capturedCommands {
		if cmd.Type == "Raw" && cmd.Args == "echo hello" {
			foundRaw = true
			break
		}
	}
	assert.True(t, foundRaw, "should have parsed raw instruction")

	// Test Ctrl instruction
	err = svc.Execute(context.Background(), CaptureOptions{
		Args: []string{"output.png", "c:c"},
	})
	require.NoError(t, err)

	var foundCtrl bool
	for _, cmd := range capturedCommands {
		if cmd.Type == "Ctrl" && cmd.Args == "c" {
			foundCtrl = true
			break
		}
	}
	assert.True(t, foundCtrl, "should have parsed ctrl instruction")

	// Test key instruction
	err = svc.Execute(context.Background(), CaptureOptions{
		Args: []string{"output.png", "k:enter"},
	})
	require.NoError(t, err)

	var foundKey bool
	for _, cmd := range capturedCommands {
		if cmd.Type == "Key" && cmd.Args == "enter" {
			foundKey = true
			break
		}
	}
	assert.True(t, foundKey, "should have parsed key instruction")
}

func TestCaptureService_Execute_WithCwd(t *testing.T) {
	t.Parallel()

	var capturedCommands []ports.CaptureCommand
	renderer := mocks.NewRenderer(t)
	renderer.On("Render", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			capturedCommands = args.Get(2).([]ports.CaptureCommand)
		}).
		Return("", nil)

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/tmp/test", nil)
	fs.On("EnsureDir", mock.Anything).Return(nil).Maybe()
	fs.On("Stat", mock.Anything).Return(&mockFileInfo{isDir: true}, nil).Maybe()
	fs.On("WriteFileAtomic", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", mock.Anything).
		Return("/tmp/test", ports.UnsareportConfig{}, false, nil)
	cfg.On("ReadConfig", mock.Anything).
		Return(ports.UnsareportConfig{
			Capture: ports.CaptureConfig{Columns: 120, Prompt: "❯ ", Colors: map[string]string{"prompt": "32"}},
		}, true, nil)

	var stdout bytes.Buffer
	svc := NewCaptureService(
		WithCaptureRenderer(renderer),
		WithCaptureFS(fs),
		WithCaptureConfig(cfg),
		WithCaptureStdout(&stdout),
		WithCaptureStderr(io.Discard),
	)

	err := svc.Execute(context.Background(), CaptureOptions{
		Cwd:  "/some/dir",
		Args: []string{"output.png", "ls"},
	})
	require.NoError(t, err)

	require.True(t, len(capturedCommands) > 3)
	assert.Equal(t, "Type", capturedCommands[0].Type)
	assert.Contains(t, capturedCommands[0].Args, "cd ")
}

func TestShellQuotePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple path", "/home/user", "'/home/user'"},
		{"path with spaces", "/home/my dir", "'/home/my dir'"},
		{"path with single quote", "/home/user's", "'/home/user'\\''s'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := shellQuotePath(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

type mockFileInfo struct {
	isDir bool
}

func (m *mockFileInfo) Name() string       { return "" }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return 0 }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }
