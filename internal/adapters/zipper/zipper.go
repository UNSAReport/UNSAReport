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

	var files []string
	err = filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
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
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk dir: %w", err)
	}

	return a.ArchiveFiles(zipPath, srcDir, files)
}

func (a *Adapter) ArchiveFiles(zipPath, baseDir string, files []string) error {
	if err := os.MkdirAll(filepath.Dir(zipPath), 0o755); err != nil {
		return fmt.Errorf("mkdir for zip: %w", err)
	}

	out, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("create zip file: %w", err)
	}
	defer out.Close() //nolint:errcheck // file close

	zw := zip.NewWriter(out)
	defer zw.Close() //nolint:errcheck // zip writer close

	for _, rel := range files {
		path := filepath.Join(baseDir, rel)
		rel = filepath.ToSlash(rel)
		rel = strings.TrimPrefix(rel, "./")

		fi, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}
		hdr, err := zip.FileInfoHeader(fi)
		if err != nil {
			return fmt.Errorf("zip header for %s: %w", path, err)
		}
		hdr.Name = rel
		hdr.Method = zip.Deflate

		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return fmt.Errorf("create zip entry for %s: %w", path, err)
		}
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open %s: %w", path, err)
		}
		_, err = io.Copy(w, f)
		f.Close() //nolint:errcheck // file close after copy
		if err != nil {
			return fmt.Errorf("copy %s to zip: %w", path, err)
		}
	}

	return nil
}
