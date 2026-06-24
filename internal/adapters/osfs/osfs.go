package osfs

import (
	"crypto/sha256"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/UNSAReport/UNSAReport/internal/ports"
	"github.com/samber/oops"
)

var _ ports.FileSystem = (*Adapter)(nil)

// Adapter implements ports.FileSystem using the OS filesystem directly.
type Adapter struct{}

// New returns a new Adapter for OS filesystem operations.
func New() *Adapter {
	return &Adapter{}
}

// EnsureDir creates the directory at path and any necessary parents with 0o755 permissions.
func (a *Adapter) EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

// FileExists reports whether a file or directory exists at path.
func (a *Adapter) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadFile returns the contents of the file at path.
func (a *Adapter) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, oops.With("path", path).Wrapf(err, "read file")
	}
	return data, nil
}

// WriteFileAtomic writes data to a temporary file and atomically renames it to path to avoid partial writes.
func (a *Adapter) WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := a.EnsureDir(dir); err != nil {
		return oops.With("path", path).Wrapf(err, "ensure dir")
	}
	tmp, err := os.CreateTemp(dir, ".tmp-unsarep-*")
	if err != nil {
		return oops.With("path", path).Wrapf(err, "create temp file")
	}
	tmpName := tmp.Name()
	defer func() {
		if err := os.Remove(tmpName); err != nil {
			slog.Warn("failed to remove temp file", "path", tmpName, "error", err)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		if closeErr := tmp.Close(); closeErr != nil {
			slog.Warn("failed to close temp file", "path", tmpName, "error", closeErr)
		}
		return oops.With("path", path).Wrapf(err, "write temp file")
	}
	if err := tmp.Close(); err != nil {
		return oops.With("path", path).Wrapf(err, "close temp file")
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return oops.With("path", path).Wrapf(err, "chmod temp file")
	}
	if err := os.Rename(tmpName, path); err != nil {
		return oops.With("path", path).Wrapf(err, "rename temp file")
	}
	return nil
}

// CopyFile copies the file at src to dst with the given permissions, creating parent directories as needed.
func (a *Adapter) CopyFile(src string, dst string, perm os.FileMode) error {
	if err := a.EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := in.Close(); err != nil {
			slog.Warn("failed to close source file", "path", src, "error", err)
		}
	}()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer func() {
		if err := out.Close(); err != nil {
			slog.Warn("failed to close destination file", "path", dst, "error", err)
		}
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// SameContent reports whether b1 and b2 have the same SHA-256 hash.
func (a *Adapter) SameContent(b1, b2 []byte) bool {
	h1 := sha256.Sum256(b1)
	h2 := sha256.Sum256(b2)
	return h1 == h2
}

// ReadDir returns the directory entries for dirname.
func (a *Adapter) ReadDir(dirname string) ([]os.DirEntry, error) {
	return os.ReadDir(dirname)
}

// Chdir changes the current working directory to dir.
func (a *Adapter) Chdir(dir string) error {
	return os.Chdir(dir)
}

// Getwd returns the current working directory.
func (a *Adapter) Getwd() (string, error) {
	return os.Getwd()
}

// Remove deletes the file or empty directory at path.
func (a *Adapter) Remove(path string) error {
	return os.Remove(path)
}

// Stat returns file info for the named path.
func (a *Adapter) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}
