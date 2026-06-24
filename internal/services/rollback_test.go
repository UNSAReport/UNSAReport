package services

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/UNSAReport/UNSAReport/internal/mocks"
	"github.com/UNSAReport/UNSAReport/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRollbackService_CreateBackup(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	entries := []Entry{
		{Kind: KindFile, Src: "report.typ", Dest: "report.typ"},
		{Kind: KindDir, Dest: "images"},
	}

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "report.typ"), []byte("original"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "images"), 0o755))

	fs := mocks.NewFileSystem(t)
	fs.On("EnsureDir", mock.Anything).Return(nil).Maybe()
	fs.On("FileExists", mock.Anything).Return(true).Maybe()
	fs.On("ReadFile", mock.Anything).Return([]byte("original"), nil).Maybe()
	fs.On("WriteFileAtomic", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	fs.On("Remove", mock.Anything).Return(nil).Maybe()

	cfg := mocks.NewConfigStore(t)

	var stdout bytes.Buffer
	svc := NewRollbackService(fs, cfg, &stdout, io.Discard)

	cfgData := ports.UnsareportConfig{TemplateVersion: "1.0.0"}
	err := svc.CreateBackup(tmpDir, entries, cfgData)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Backup created:")
}

func TestRollbackService_Rollback(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create backup using real filesystem
	backupDirPath := filepath.Join(tmpDir, backupDir)
	require.NoError(t, os.MkdirAll(backupDirPath, 0o755))

	manifestJSON := `{
		"timestamp": "2026-01-01T00:00:00Z",
		"template_version": "1.0.0",
		"files": [
			{"relative_path": "report.typ", "original_path": "` + filepath.Join(tmpDir, "report.typ") + `"}
		]
	}`
	require.NoError(t, os.WriteFile(filepath.Join(backupDirPath, "manifest.json"), []byte(manifestJSON), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(backupDirPath, "report.typ"), []byte("backed up content"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "report.typ"), []byte("current content"), 0o644))

	// Use real fs adapter for reading, mock for writing
	rollbackFS := &rollbackTestFS{
		manifestJSON: manifestJSON,
		backupData:   []byte("backed up content"),
	}

	var stdout bytes.Buffer
	svc := NewRollbackService(rollbackFS, mocks.NewConfigStore(t), &stdout, io.Discard)

	err := svc.Rollback(tmpDir)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Rollback complete:")
}

// rollbackTestFS is a minimal filesystem mock for rollback tests that reads from real FS
// but uses mock for write operations.
type rollbackTestFS struct {
	manifestJSON string
	backupData   []byte
}

func (f *rollbackTestFS) EnsureDir(path string) error          { return os.MkdirAll(path, 0o755) }
func (f *rollbackTestFS) FileExists(path string) bool          { _, err := os.Stat(path); return err == nil }
func (f *rollbackTestFS) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }
func (f *rollbackTestFS) WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}
func (f *rollbackTestFS) CopyFile(src, dst string, perm os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, perm)
}
func (f *rollbackTestFS) SameContent(a, b []byte) bool                  { return string(a) == string(b) }
func (f *rollbackTestFS) ReadDir(dirname string) ([]os.DirEntry, error) { return os.ReadDir(dirname) }
func (f *rollbackTestFS) Chdir(dir string) error                        { return os.Chdir(dir) }
func (f *rollbackTestFS) Getwd() (string, error)                        { return os.Getwd() }
func (f *rollbackTestFS) Remove(path string) error                      { return os.Remove(path) }
func (f *rollbackTestFS) Stat(name string) (os.FileInfo, error)         { return os.Stat(name) }

func TestRollbackService_Rollback_NoBackup(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	fs := mocks.NewFileSystem(t)
	fs.On("FileExists", mock.Anything).Return(false)

	cfg := mocks.NewConfigStore(t)

	var stdout bytes.Buffer
	svc := NewRollbackService(fs, cfg, &stdout, io.Discard)

	err := svc.Rollback(tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no backup found")
}

func TestRollbackService_HasBackup(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	backupDirPath := filepath.Join(tmpDir, backupDir)

	fs := mocks.NewFileSystem(t)
	manifestPath := filepath.Join(backupDirPath, "manifest.json")
	fs.On("FileExists", manifestPath).Return(true)
	fs.On("FileExists", mock.AnythingOfType("string")).Return(false)

	cfg := mocks.NewConfigStore(t)

	svc := NewRollbackService(fs, cfg, io.Discard, io.Discard)
	assert.True(t, svc.HasBackup(tmpDir))
}
