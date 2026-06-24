package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFetcher struct {
	rawFunc func(repo, ref, path string) ([]byte, error)
}

func (m *mockFetcher) Fetch(_ context.Context, repo, ref, templatePath string) (map[string][]byte, error) {
	return nil, nil
}

func (m *mockFetcher) FetchRaw(_ context.Context, repo, ref, path string) ([]byte, error) {
	if m.rawFunc != nil {
		return m.rawFunc(repo, ref, path)
	}
	return nil, nil
}

func (m *mockFetcher) LoadLocal(dir string) (map[string][]byte, error) {
	return nil, nil
}

func TestLocalAdapter_ListTemplates_Success(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	reg := registryTemplateFile{
		Templates: map[string]registryTemplateEntry{
			"report-a": {
				Description: "First template",
				DistTags:    map[string]string{"latest": "1.0.0"},
				Versions: map[string]registryTemplateVersion{
					"1.0.0": {Path: "report-a/1.0.0"},
				},
			},
			"report-b": {
				Description: "Second template",
				DistTags:    map[string]string{"latest": "2.0.0"},
				Versions: map[string]registryTemplateVersion{
					"2.0.0": {Path: "report-b/2.0.0"},
				},
			},
		},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "registry.json"), data, 0o644))

	a := New(tmpDir)
	templates, err := a.ListTemplates()
	require.NoError(t, err)
	assert.Len(t, templates, 2)
}

func TestLocalAdapter_ListTemplates_Empty(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	reg := registryTemplateFile{
		Templates: map[string]registryTemplateEntry{},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "registry.json"), data, 0o644))

	a := New(tmpDir)
	_, err = a.ListTemplates()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no templates found")
}

func TestLocalAdapter_ListTemplates_NoRegistryFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	a := New(tmpDir)
	_, err := a.ListTemplates()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no templates found")
}

func TestLocalAdapter_GetTemplate_Success(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	reg := registryTemplateFile{
		Templates: map[string]registryTemplateEntry{
			"mytemplate": {
				Description: "My template",
				DistTags:    map[string]string{"latest": "1.0.0"},
				Versions: map[string]registryTemplateVersion{
					"1.0.0": {Path: "mytemplate/1.0.0"},
				},
			},
		},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "registry.json"), data, 0o644))

	a := New(tmpDir)
	tmpl, err := a.GetTemplate("mytemplate")
	require.NoError(t, err)
	assert.Equal(t, "mytemplate", tmpl.Name)
	assert.Equal(t, "My template", tmpl.Description)
}

func TestLocalAdapter_GetTemplate_NotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	reg := registryTemplateFile{
		Templates: map[string]registryTemplateEntry{
			"existing": {
				Description: "Exists",
				DistTags:    map[string]string{"latest": "1.0.0"},
				Versions: map[string]registryTemplateVersion{
					"1.0.0": {Path: "existing/1.0.0"},
				},
			},
		},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "registry.json"), data, 0o644))

	a := New(tmpDir)
	_, err = a.GetTemplate("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLocalAdapter_GetTemplateVersion_Success(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	reg := registryTemplateFile{
		Templates: map[string]registryTemplateEntry{
			"mytemplate": {
				Description: "My template",
				DistTags: map[string]string{
					"latest": "2.0.0",
					"stable": "1.0.0",
				},
				Versions: map[string]registryTemplateVersion{
					"1.0.0": {Path: "mytemplate/1.0.0"},
					"2.0.0": {Path: "mytemplate/2.0.0"},
				},
			},
		},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "registry.json"), data, 0o644))

	a := New(tmpDir)

	tests := []struct {
		name      string
		rangeSpec string
		wantVer   string
	}{
		{"latest", "latest", "2.0.0"},
		{"caret range", "^1.0.0", "1.0.0"},
		{"exact version", "2.0.0", "2.0.0"},
		{"tilde range", "~1.0.0", "1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmpl, err := a.GetTemplateVersion("mytemplate", tt.rangeSpec)
			require.NoError(t, err)
			assert.Equal(t, tt.wantVer, tmpl.Version)
		})
	}
}

func TestLocalAdapter_GetTemplateVersion_NotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	reg := registryTemplateFile{
		Templates: map[string]registryTemplateEntry{},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "registry.json"), data, 0o644))

	a := New(tmpDir)
	_, err = a.GetTemplateVersion("nonexistent", "latest")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLocalAdapter_GetTemplateVersion_NoMatchingVersion(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	reg := registryTemplateFile{
		Templates: map[string]registryTemplateEntry{
			"mytemplate": {
				Description: "My template",
				DistTags:    map[string]string{"latest": "1.0.0"},
				Versions: map[string]registryTemplateVersion{
					"1.0.0": {Path: "mytemplate/1.0.0"},
				},
			},
		},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "registry.json"), data, 0o644))

	a := New(tmpDir)
	_, err = a.GetTemplateVersion("mytemplate", ">=5.0.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no version matching")
}

func TestLocalAdapter_GetTemplateVersion_NoLatestDistTag(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	reg := registryTemplateFile{
		Templates: map[string]registryTemplateEntry{
			"mytemplate": {
				Description: "My template",
				DistTags:    map[string]string{},
				Versions: map[string]registryTemplateVersion{
					"1.0.0": {Path: "mytemplate/1.0.0"},
				},
			},
		},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "registry.json"), data, 0o644))

	a := New(tmpDir)
	_, err = a.GetTemplateVersion("mytemplate", "latest")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no 'latest' dist-tag found")
}

func TestLocalAdapter_GetTemplateVersion_InvalidRegistry(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "registry.json"), []byte("not json"), 0o644))

	a := New(tmpDir)
	_, err := a.GetTemplateVersion("mytemplate", "latest")
	require.Error(t, err)
}

func TestLocalAdapter_TemplateExists(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "mytemplate"), 0o755))

	a := New(tmpDir)
	assert.True(t, a.TemplateExists("mytemplate"))
	assert.False(t, a.TemplateExists("nonexistent"))
}

func TestLocalAdapter_TemplateDir(t *testing.T) {
	t.Parallel()

	a := New("/some/path")
	assert.Equal(t, "/some/path", a.TemplateDir())
}

func TestLocalAdapter_NewLocal(t *testing.T) {
	t.Parallel()

	a := NewLocal("/some/path")
	assert.Equal(t, "/some/path", a.TemplateDir())
}

func TestLocalAdapter_LoadRegistry_InvalidJSON(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "registry.json"), []byte("{invalid"), 0o644))

	a := New(tmpDir)
	_, err := a.ListTemplates()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no templates found")
}

func TestRemoteAdapter_ListTemplates_Success(t *testing.T) {
	t.Parallel()

	reg := registryTemplateFile{
		Templates: map[string]registryTemplateEntry{
			"report-a": {
				Description: "First template",
				DistTags:    map[string]string{"latest": "1.0.0"},
				Versions: map[string]registryTemplateVersion{
					"1.0.0": {Path: "report-a/1.0.0"},
				},
			},
			"report-b": {
				Description: "Second template",
				DistTags:    map[string]string{"latest": "2.0.0"},
				Versions: map[string]registryTemplateVersion{
					"2.0.0": {Path: "report-b/2.0.0"},
				},
			},
		},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)

	fetcher := &mockFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return data, nil
		},
	}

	a := NewRemote(fetcher)
	templates, err := a.ListTemplates()
	require.NoError(t, err)
	assert.Len(t, templates, 2)
}

func TestRemoteAdapter_ListTemplates_FetchError(t *testing.T) {
	t.Parallel()

	fetcher := &mockFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return nil, assert.AnError
		},
	}

	a := NewRemote(fetcher)
	_, err := a.ListTemplates()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch registry.json")
}

func TestRemoteAdapter_ListTemplates_InvalidJSON(t *testing.T) {
	t.Parallel()

	fetcher := &mockFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return []byte("not json"), nil
		},
	}

	a := NewRemote(fetcher)
	_, err := a.ListTemplates()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse registry.json")
}

func TestRemoteAdapter_GetTemplate_Success(t *testing.T) {
	t.Parallel()

	reg := registryTemplateFile{
		Templates: map[string]registryTemplateEntry{
			"mytemplate": {
				Description: "My template",
				DistTags:    map[string]string{"latest": "1.0.0"},
				Versions: map[string]registryTemplateVersion{
					"1.0.0": {Path: "mytemplate/1.0.0"},
				},
			},
		},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)

	fetcher := &mockFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return data, nil
		},
	}

	a := NewRemote(fetcher)
	tmpl, err := a.GetTemplate("mytemplate")
	require.NoError(t, err)
	assert.Equal(t, "mytemplate", tmpl.Name)
	assert.Equal(t, "My template", tmpl.Description)
}

func TestRemoteAdapter_GetTemplate_NotFound(t *testing.T) {
	t.Parallel()

	reg := registryTemplateFile{
		Templates: map[string]registryTemplateEntry{},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)

	fetcher := &mockFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return data, nil
		},
	}

	a := NewRemote(fetcher)
	_, err = a.GetTemplate("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRemoteAdapter_GetTemplateVersion_Success(t *testing.T) {
	t.Parallel()

	reg := registryTemplateFile{
		Templates: map[string]registryTemplateEntry{
			"mytemplate": {
				Description: "My template",
				DistTags: map[string]string{
					"latest": "2.0.0",
				},
				Versions: map[string]registryTemplateVersion{
					"1.0.0": {Path: "mytemplate/1.0.0"},
					"2.0.0": {Path: "mytemplate/2.0.0"},
				},
			},
		},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)

	fetcher := &mockFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return data, nil
		},
	}

	a := NewRemote(fetcher)
	tmpl, err := a.GetTemplateVersion("mytemplate", "latest")
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", tmpl.Version)
	assert.Equal(t, "mytemplate/2.0.0", tmpl.Path)
}

func TestRemoteAdapter_GetTemplateVersion_NotFound(t *testing.T) {
	t.Parallel()

	reg := registryTemplateFile{
		Templates: map[string]registryTemplateEntry{},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)

	fetcher := &mockFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return data, nil
		},
	}

	a := NewRemote(fetcher)
	_, err = a.GetTemplateVersion("nonexistent", "latest")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRemoteAdapter_GetTemplateVersion_NoMatchingVersion(t *testing.T) {
	t.Parallel()

	reg := registryTemplateFile{
		Templates: map[string]registryTemplateEntry{
			"mytemplate": {
				Description: "My template",
				DistTags:    map[string]string{"latest": "1.0.0"},
				Versions: map[string]registryTemplateVersion{
					"1.0.0": {Path: "mytemplate/1.0.0"},
				},
			},
		},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)

	fetcher := &mockFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return data, nil
		},
	}

	a := NewRemote(fetcher)
	_, err = a.GetTemplateVersion("mytemplate", ">=5.0.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no version matching")
}

func TestRemoteAdapter_GetTemplateVersion_NoLatestDistTag(t *testing.T) {
	t.Parallel()

	reg := registryTemplateFile{
		Templates: map[string]registryTemplateEntry{
			"mytemplate": {
				Description: "My template",
				DistTags:    map[string]string{},
				Versions: map[string]registryTemplateVersion{
					"1.0.0": {Path: "mytemplate/1.0.0"},
				},
			},
		},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)

	fetcher := &mockFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return data, nil
		},
	}

	a := NewRemote(fetcher)
	_, err = a.GetTemplateVersion("mytemplate", "latest")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no 'latest' dist-tag found")
}

func TestRemoteAdapter_ConvertTemplate(t *testing.T) {
	t.Parallel()

	a := NewRemote(&mockFetcher{})
	entry := registryTemplateEntry{
		Description: "Test description",
		DistTags:    map[string]string{"latest": "1.0.0"},
		Versions: map[string]registryTemplateVersion{
			"1.0.0": {Path: "test/1.0.0"},
		},
	}

	tmpl := a.convertTemplate("test", entry)
	assert.Equal(t, "test", tmpl.Name)
	assert.Equal(t, "Test description", tmpl.Description)
}

func TestRemoteAdapter_WithHTTPServer(t *testing.T) {
	t.Parallel()

	reg := registryTemplateFile{
		Templates: map[string]registryTemplateEntry{
			"remote-tpl": {
				Description: "Remote template",
				DistTags:    map[string]string{"latest": "3.0.0"},
				Versions: map[string]registryTemplateVersion{
					"3.0.0": {Path: "remote-tpl/3.0.0"},
				},
			},
		},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	defer srv.Close()

	fetcher := &mockFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return fetchHTTP(srv.URL)
		},
	}

	a := NewRemote(fetcher)
	tmpl, err := a.GetTemplateVersion("remote-tpl", "latest")
	require.NoError(t, err)
	assert.Equal(t, "3.0.0", tmpl.Version)
}
