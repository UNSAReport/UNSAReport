//go:build integration

package zipper

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZipper_ArchiveDir(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create source files
	srcDir := filepath.Join(tmpDir, "src")
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "sub", "helper.go"), []byte("package sub"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "README.md"), []byte("# Test"), 0o644))

	zipPath := filepath.Join(tmpDir, "output.zip")
	a := New()
	err := a.ArchiveDir(zipPath, srcDir)
	require.NoError(t, err)

	// Verify the zip was created and contains expected files
	assert.FileExists(t, zipPath)

	zr, err := zip.OpenReader(zipPath)
	require.NoError(t, err)
	defer zr.Close()

	files := make(map[string]bool)
	for _, f := range zr.File {
		if !f.FileInfo().IsDir() {
			files[f.Name] = true
		}
	}

	assert.True(t, files["main.go"], "should contain main.go")
	assert.True(t, files["sub/helper.go"], "should contain sub/helper.go")
	assert.True(t, files["README.md"], "should contain README.md")
	assert.Equal(t, 3, len(files), "should contain exactly 3 files")
}

func TestZipper_ArchiveDir_MissingSource(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	a := New()
	err := a.ArchiveDir(filepath.Join(tmpDir, "out.zip"), filepath.Join(tmpDir, "nonexistent"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSourceMissing)
}

func TestZipper_ArchiveDir_SourceNotDir(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	file := filepath.Join(tmpDir, "file.txt")
	require.NoError(t, os.WriteFile(file, []byte("not a dir"), 0o644))

	a := New()
	err := a.ArchiveDir(filepath.Join(tmpDir, "out.zip"), file)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

func TestZipper_ArchiveDir_EmptyDir(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "empty")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	zipPath := filepath.Join(tmpDir, "empty.zip")
	a := New()
	err := a.ArchiveDir(zipPath, srcDir)
	require.NoError(t, err)

	// Empty zip should still be valid
	zr, err := zip.OpenReader(zipPath)
	require.NoError(t, err)
	defer zr.Close()
	assert.Empty(t, zr.File)
}

func TestZipper_ArchiveFiles(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create source files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("content a"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("content b"), 0o644))

	zipPath := filepath.Join(tmpDir, "output.zip")
	a := New()
	err := a.ArchiveFiles(zipPath, tmpDir, []string{"a.txt", "b.txt"})
	require.NoError(t, err)

	// Verify zip contents
	zr, err := zip.OpenReader(zipPath)
	require.NoError(t, err)
	defer zr.Close()

	require.Len(t, zr.File, 2)

	// Read and verify file contents
	for _, f := range zr.File {
		rc, err := f.Open()
		require.NoError(t, err)
		data, err := io.ReadAll(rc)
		rc.Close()
		require.NoError(t, err)

		switch f.Name {
		case "a.txt":
			assert.Equal(t, "content a", string(data))
		case "b.txt":
			assert.Equal(t, "content b", string(data))
		default:
			t.Errorf("unexpected file in zip: %s", f.Name)
		}
	}
}

func TestZipper_ArchiveFiles_NestedPaths(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "dir", "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "dir", "sub", "deep.txt"), []byte("deep"), 0o644))

	zipPath := filepath.Join(tmpDir, "nested.zip")
	a := New()
	err := a.ArchiveFiles(zipPath, tmpDir, []string{"dir/sub/deep.txt"})
	require.NoError(t, err)

	zr, err := zip.OpenReader(zipPath)
	require.NoError(t, err)
	defer zr.Close()

	require.Len(t, zr.File, 1)
	assert.Equal(t, "dir/sub/deep.txt", zr.File[0].Name)

	rc, err := zr.File[0].Open()
	require.NoError(t, err)
	data, err := io.ReadAll(rc)
	rc.Close()
	require.NoError(t, err)
	assert.Equal(t, "deep", string(data))
}

func TestZipper_ArchiveDir_ContentType(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file.go"), []byte("package main\n"), 0o644))

	zipPath := filepath.Join(tmpDir, "test.zip")
	a := New()
	require.NoError(t, a.ArchiveDir(zipPath, srcDir))

	// Verify the zip uses Deflate compression (not Store)
	zr, err := zip.OpenReader(zipPath)
	require.NoError(t, err)
	defer zr.Close()

	require.Len(t, zr.File, 1)
	assert.Equal(t, zip.Deflate, zr.File[0].Method)
}
