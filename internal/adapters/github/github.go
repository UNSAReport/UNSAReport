package github

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	httpTimeout       = 30 * time.Second
	maxBodySize int64 = 50 << 20 // 50MB
)

var httpClient = &http.Client{Timeout: httpTimeout}

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Fetch(ctx context.Context, repo, ref, templatePath string) (map[string][]byte, error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid repo %q (expected owner/repo)", repo)
	}
	owner, name := parts[0], parts[1]

	zipURL := fmt.Sprintf("https://codeload.github.com/%s/%s/zip/%s", owner, name, ref)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, zipURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "unsarep")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // HTTP response body close
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return nil, fmt.Errorf("failed to fetch templates: %s (%s)", resp.Status, strings.TrimSpace(string(b)))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("read zip: %w", err)
	}

	prefix := name + "-" + ref + "/" + templatePath + "/"

	out := make(map[string][]byte)
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}
		rel := strings.TrimPrefix(f.Name, prefix)
		if rel == "" {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open file in zip: %w", err)
		}
		data, err := io.ReadAll(rc)
		if err := rc.Close(); err != nil {
			return nil, fmt.Errorf("close file in zip: %w", err)
		}
		if err != nil {
			return nil, fmt.Errorf("read file in zip: %w", err)
		}
		out[rel] = data
	}

	return out, nil
}

func (a *Adapter) LoadLocal(dir string) (map[string][]byte, error) {
	out := make(map[string][]byte)
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("abs dir: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat dir: %w", err)
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
			return fmt.Errorf("read file: %w", err)
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

	clean := make(map[string][]byte)
	for k, v := range out {
		clean[strings.TrimPrefix(k, "./")] = v
	}

	return clean, nil
}

func (a *Adapter) FetchRaw(ctx context.Context, repo, ref, path string) ([]byte, error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid repo %q (expected owner/repo)", repo)
	}
	owner, name := parts[0], parts[1]

	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/refs/heads/%s/%s", owner, name, ref, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "unsarep")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // HTTP response body close
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return nil, fmt.Errorf("failed to fetch %s: %s (%s)", path, resp.Status, strings.TrimSpace(string(b)))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return body, nil
}
