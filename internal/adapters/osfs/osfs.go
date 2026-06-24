package osfs

import (
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
)

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func (a *Adapter) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (a *Adapter) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (a *Adapter) WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := a.EnsureDir(dir); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-unsarep-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) //nolint:errcheck // best-effort cleanup

	if _, err := tmp.Write(data); err != nil {
		tmp.Close() //nolint:errcheck // best-effort cleanup
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func (a *Adapter) CopyFile(src string, dst string, perm os.FileMode) error {
	if err := a.EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck // file close
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer out.Close() //nolint:errcheck // file close
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func (a *Adapter) SameContent(b1, b2 []byte) bool {
	h1 := sha256.Sum256(b1)
	h2 := sha256.Sum256(b2)
	return h1 == h2
}

func (a *Adapter) ReadDir(dirname string) ([]os.DirEntry, error) {
	return os.ReadDir(dirname)
}

func (a *Adapter) Chdir(dir string) error {
	return os.Chdir(dir)
}

func (a *Adapter) Getwd() (string, error) {
	return os.Getwd()
}

func (a *Adapter) Remove(path string) error {
	return os.Remove(path)
}

func (a *Adapter) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}
