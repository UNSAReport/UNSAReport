//go:build integration

package osfs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOSFS_EnsureDir(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	a := New()

	dir := filepath.Join(tmpDir, "a", "b", "c")
	err := a.EnsureDir(dir)
	require.NoError(t, err)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestOSFS_FileExists(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	a := New()

	file := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello"), 0o644))

	assert.True(t, a.FileExists(file))
	assert.False(t, a.FileExists(filepath.Join(tmpDir, "nonexistent")))
}

func TestOSFS_ReadFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	a := New()

	file := filepath.Join(tmpDir, "test.txt")
	expected := []byte("hello world")
	require.NoError(t, os.WriteFile(file, expected, 0o644))

	data, err := a.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, expected, data)
}

func TestOSFS_ReadFile_NotFound(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	a := New()

	_, err := a.ReadFile(filepath.Join(tmpDir, "nonexistent"))
	require.Error(t, err)
}

func TestOSFS_WriteFileAtomic(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	a := New()

	file := filepath.Join(tmpDir, "atomic.txt")
	data := []byte("atomic write")

	err := a.WriteFileAtomic(file, data, 0o644)
	require.NoError(t, err)

	written, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, data, written)
}

func TestOSFS_CopyFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	a := New()

	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst", "dst.txt")
	data := []byte("copy me")

	require.NoError(t, os.WriteFile(src, data, 0o644))

	err := a.CopyFile(src, dst, 0o644)
	require.NoError(t, err)

	written, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, data, written)
}

func TestOSFS_SameContent(t *testing.T) {
	t.Parallel()
	a := New()

	assert.True(t, a.SameContent([]byte("hello"), []byte("hello")))
	assert.False(t, a.SameContent([]byte("hello"), []byte("world")))
	assert.True(t, a.SameContent([]byte{}, []byte{}))
}

func TestOSFS_ReadDir(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	a := New()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("b"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "sub"), 0o755))

	entries, err := a.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestOSFS_Chdir_Getwd(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	a := New()

	origDir, err := a.Getwd()
	require.NoError(t, err)
	defer a.Chdir(origDir)

	err = a.Chdir(tmpDir)
	require.NoError(t, err)

	cwd, err := a.Getwd()
	require.NoError(t, err)
	assert.Equal(t, tmpDir, cwd)
}

func TestOSFS_Remove(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	a := New()

	file := filepath.Join(tmpDir, "to_delete.txt")
	require.NoError(t, os.WriteFile(file, []byte("delete me"), 0o644))

	err := a.Remove(file)
	require.NoError(t, err)
	assert.False(t, a.FileExists(file))
}

func TestOSFS_Remove_Nonexistent(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	a := New()

	err := a.Remove(filepath.Join(tmpDir, "nonexistent"))
	require.Error(t, err)
}

func TestOSFS_Stat(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	a := New()

	file := filepath.Join(tmpDir, "stat.txt")
	require.NoError(t, os.WriteFile(file, []byte("stat me"), 0o644))

	info, err := a.Stat(file)
	require.NoError(t, err)
	assert.Equal(t, "stat.txt", info.Name())
	assert.False(t, info.IsDir())

	dirInfo, err := a.Stat(tmpDir)
	require.NoError(t, err)
	assert.True(t, dirInfo.IsDir())
}
