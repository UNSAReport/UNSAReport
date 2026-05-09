package zipper

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var ErrSourceMissing = errors.New("source directory not found")

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) ArchiveDir(zipPath, srcDir string) error {
	st, err := os.Stat(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrSourceMissing
		}
		return fmt.Errorf("stat source dir: %w", err)
	}
	if !st.IsDir() {
		return fmt.Errorf("%s is not a directory", srcDir)
	}

	if err := os.MkdirAll(filepath.Dir(zipPath), 0o755); err != nil {
		return fmt.Errorf("mkdir for zip: %w", err)
	}

	out, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("create zip file: %w", err)
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		rel = strings.TrimPrefix(rel, "./")

		fi, err := d.Info()
		if err != nil {
			return err
		}
		hdr, err := zip.FileInfoHeader(fi)
		if err != nil {
			return err
		}
		hdr.Name = rel
		hdr.Method = zip.Deflate

		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
}
