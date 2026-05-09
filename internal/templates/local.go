package templates

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func LoadFromDir(dir string) (Files, error) {
	out := make(Files)
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fs.ErrInvalid
	}

	err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(abs, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		out[rel] = b
		return nil
	})
	if err != nil {
		return nil, err
	}

	clean := make(Files)
	for k, v := range out {
		clean[strings.TrimPrefix(k, "./")] = v
	}

	return clean, nil
}
