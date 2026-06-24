package ports

import (
	"os"
)

// FileSystem abstracts filesystem operations to enable testing and alternative implementations.
type FileSystem interface {
	EnsureDir(path string) error
	FileExists(path string) bool
	ReadFile(path string) ([]byte, error)
	WriteFileAtomic(path string, data []byte, perm os.FileMode) error
	CopyFile(src string, dst string, perm os.FileMode) error
	SameContent(a, b []byte) bool
	ReadDir(dirname string) ([]os.DirEntry, error)
	Chdir(dir string) error
	Getwd() (string, error)
	Remove(path string) error
	Stat(name string) (os.FileInfo, error)
}
