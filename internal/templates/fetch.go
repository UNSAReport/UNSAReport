package templates

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Files map[string][]byte

func Fetch(ctx context.Context, src Source) (Files, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.ZipURL(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "lab-report")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return nil, fmt.Errorf("failed to fetch templates: %s (%s)", resp.Status, strings.TrimSpace(string(b)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}

	prefix, err := findTemplatePrefix(zr)
	if err != nil {
		return nil, err
	}

	out := make(Files)
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
			return nil, err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		out[rel] = data
	}

	return out, nil
}

func findTemplatePrefix(zr *zip.Reader) (string, error) {
	for _, f := range zr.File {
		if idx := strings.Index(f.Name, "/template/"); idx != -1 {
			root := f.Name[:idx+1]
			return root + "template/", nil
		}
	}
	return "", fmt.Errorf("template/ directory not found in repository archive")
}
