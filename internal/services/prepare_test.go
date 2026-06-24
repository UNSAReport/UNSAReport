package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/UNSAReport/UNSAReport/internal/mocks"
	"github.com/UNSAReport/UNSAReport/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPrepareService_Execute_NoProjectRoot(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/tmp/test", nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", mock.Anything).
		Return("/tmp/test", ports.UnsareportConfig{}, false, nil)

	var stdout bytes.Buffer
	svc := NewPrepareService(
		WithPrepareCompiler(mocks.NewCompiler(t)),
		WithPrepareArchiver(mocks.NewArchiver(t)),
		WithPrepareFS(fs),
		WithPrepareConfig(cfg),
		WithPrepareStdout(&stdout),
		WithPrepareStderr(io.Discard),
	)

	err := svc.Execute(context.Background(), PrepareOptions{}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsareport.json not found")
}

func TestPrepareService_Execute_MissingReportFile(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/tmp/test", nil)
	fs.On("Chdir", mock.Anything).Return(nil)
	fs.On("FileExists", mock.Anything).Return(false)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", mock.Anything).
		Return("/tmp/test", ports.UnsareportConfig{
			Mode: "single",
			Prepare: ports.PrepareConfig{
				Input:  ports.PrepareInputConfig{SrcDir: "src", ReportFile: "report.typ"},
				Output: ports.PrepareOutputConfig{SubmissionDir: "submission", FileTemplate: "{output_type}_{lab_number}"},
			},
		}, true, nil)

	var stdout bytes.Buffer
	svc := NewPrepareService(
		WithPrepareCompiler(mocks.NewCompiler(t)),
		WithPrepareArchiver(mocks.NewArchiver(t)),
		WithPrepareFS(fs),
		WithPrepareConfig(cfg),
		WithPrepareStdout(&stdout),
		WithPrepareStderr(io.Discard),
	)

	err := svc.Execute(context.Background(), PrepareOptions{}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "report file not found")
}

func TestPrepareService_Execute_CompilerQueryVarsError(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/tmp/test", nil)
	fs.On("Chdir", mock.Anything).Return(nil)
	fs.On("FileExists", mock.Anything).Return(true)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", mock.Anything).
		Return("/tmp/test", ports.UnsareportConfig{
			Mode: "single",
			Prepare: ports.PrepareConfig{
				Input:  ports.PrepareInputConfig{SrcDir: "src", ReportFile: "report.typ"},
				Output: ports.PrepareOutputConfig{SubmissionDir: "submission", FileTemplate: "{output_type}_{lab_number}"},
			},
		}, true, nil)

	compiler := mocks.NewCompiler(t)
	compiler.On("QueryVars", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("typst query failed"))

	var stdout bytes.Buffer
	svc := NewPrepareService(
		WithPrepareCompiler(compiler),
		WithPrepareArchiver(mocks.NewArchiver(t)),
		WithPrepareFS(fs),
		WithPrepareConfig(cfg),
		WithPrepareStdout(&stdout),
		WithPrepareStderr(io.Discard),
	)

	err := svc.Execute(context.Background(), PrepareOptions{}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query vars")
}

func TestPrepareService_Execute_CompilerCompileError(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/tmp/test", nil)
	fs.On("Chdir", mock.Anything).Return(nil)
	fs.On("FileExists", mock.Anything).Return(true)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", mock.Anything).
		Return("/tmp/test", ports.UnsareportConfig{
			Mode: "single",
			Prepare: ports.PrepareConfig{
				Input:  ports.PrepareInputConfig{SrcDir: "src", ReportFile: "report.typ"},
				Output: ports.PrepareOutputConfig{SubmissionDir: "submission", FileTemplate: "{output_type}_{lab_number}"},
			},
		}, true, nil)

	compiler := mocks.NewCompiler(t)
	compiler.On("QueryVars", mock.Anything, mock.Anything).
		Return(map[string]string{"lab_number": "1"}, nil)
	compiler.On("Compile", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(fmt.Errorf("typst compile error: missing import"))

	var stdout bytes.Buffer
	svc := NewPrepareService(
		WithPrepareCompiler(compiler),
		WithPrepareArchiver(mocks.NewArchiver(t)),
		WithPrepareFS(fs),
		WithPrepareConfig(cfg),
		WithPrepareStdout(&stdout),
		WithPrepareStderr(io.Discard),
	)

	err := svc.Execute(context.Background(), PrepareOptions{}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compile report")
}

func TestPrepareService_Execute_MultiModeNoLabDir(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/tmp/test", nil)
	fs.On("Chdir", mock.Anything).Return(nil).Maybe()
	fs.On("FileExists", mock.Anything).Return(true).Maybe()

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", mock.Anything).
		Return("/tmp/test", ports.UnsareportConfig{
			Mode:     "multi",
			Sessions: []string{"l1", "l2"},
			Prepare: ports.PrepareConfig{
				Input:  ports.PrepareInputConfig{SrcDir: "src", ReportFile: "report.typ"},
				Output: ports.PrepareOutputConfig{SubmissionDir: "submission", FileTemplate: "{output_type}_{lab_number}"},
			},
		}, true, nil)

	var stdout bytes.Buffer
	svc := NewPrepareService(
		WithPrepareCompiler(mocks.NewCompiler(t)),
		WithPrepareArchiver(mocks.NewArchiver(t)),
		WithPrepareFS(fs),
		WithPrepareConfig(cfg),
		WithPrepareStdout(&stdout),
		WithPrepareStderr(io.Discard),
	)

	err := svc.Execute(context.Background(), PrepareOptions{}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "you must either provide a lab directory")
}

func TestPrepareService_Execute_MultiModeInvalidSession(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/tmp/test/l3", nil)
	fs.On("Chdir", mock.Anything).Return(nil).Maybe()
	fs.On("FileExists", mock.Anything).Return(true).Maybe()

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", mock.Anything).
		Return("/tmp/test", ports.UnsareportConfig{
			Mode:     "multi",
			Sessions: []string{"l1", "l2"},
			Prepare: ports.PrepareConfig{
				Input:  ports.PrepareInputConfig{SrcDir: "src", ReportFile: "report.typ"},
				Output: ports.PrepareOutputConfig{SubmissionDir: "submission"},
			},
		}, true, nil)

	var stdout bytes.Buffer
	svc := NewPrepareService(
		WithPrepareCompiler(mocks.NewCompiler(t)),
		WithPrepareArchiver(mocks.NewArchiver(t)),
		WithPrepareFS(fs),
		WithPrepareConfig(cfg),
		WithPrepareStdout(&stdout),
		WithPrepareStderr(io.Discard),
	)

	err := svc.Execute(context.Background(), PrepareOptions{}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not registered")
}

func TestPrepareService_Execute_LabArgOnSingleMode(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/tmp/test", nil)
	fs.On("Chdir", mock.Anything).Return(nil).Maybe()
	fs.On("FileExists", mock.Anything).Return(true).Maybe()

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", mock.Anything).
		Return("/tmp/test", ports.UnsareportConfig{
			Mode: "single",
			Prepare: ports.PrepareConfig{
				Input:  ports.PrepareInputConfig{SrcDir: "src", ReportFile: "report.typ"},
				Output: ports.PrepareOutputConfig{SubmissionDir: "submission"},
			},
		}, true, nil)

	var stdout bytes.Buffer
	svc := NewPrepareService(
		WithPrepareCompiler(mocks.NewCompiler(t)),
		WithPrepareArchiver(mocks.NewArchiver(t)),
		WithPrepareFS(fs),
		WithPrepareConfig(cfg),
		WithPrepareStdout(&stdout),
		WithPrepareStderr(io.Discard),
	)

	err := svc.Execute(context.Background(), PrepareOptions{}, "l1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lab argument provided but template is not multi-mode")
}
