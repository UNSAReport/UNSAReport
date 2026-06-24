package registry

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/UNSAReport/UNSAReport/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCompFetcher struct {
	rawFunc func(repo, ref, path string) ([]byte, error)
}

func (m *mockCompFetcher) Fetch(_ context.Context, repo, ref, templatePath string) (map[string][]byte, error) {
	return nil, nil
}

func (m *mockCompFetcher) FetchRaw(_ context.Context, repo, ref, path string) ([]byte, error) {
	if m.rawFunc != nil {
		return m.rawFunc(repo, ref, path)
	}
	return nil, nil
}

func (m *mockCompFetcher) LoadLocal(dir string) (map[string][]byte, error) {
	return nil, nil
}

func fetchHTTP(srvURL string) ([]byte, error) {
	resp, err := http.Get(srvURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return io.ReadAll(resp.Body)
}

func TestComponentRegistryAdapter_ListComponents_Success(t *testing.T) {
	t.Parallel()

	reg := registryFile{
		Components: map[string]registryComponentEntry{
			"comp-a": {
				Description: "Component A",
				DistTags:    map[string]string{"latest": "1.0.0"},
				Versions: map[string]registryVersionEntry{
					"1.0.0": {Path: "comp-a/1.0.0.typ", Dependencies: nil},
				},
			},
			"comp-b": {
				Description: "Component B",
				DistTags:    map[string]string{"latest": "2.0.0"},
				Versions: map[string]registryVersionEntry{
					"2.0.0": {Path: "comp-b/2.0.0.typ", Dependencies: []string{"comp-a@^1.0.0"}},
				},
			},
		},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)

	fetcher := &mockCompFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return data, nil
		},
	}

	a := NewComponentRegistry(fetcher)
	components, err := a.ListComponents()
	require.NoError(t, err)
	assert.Len(t, components, 2)
}

func TestComponentRegistryAdapter_ListComponents_FetchError(t *testing.T) {
	t.Parallel()

	fetcher := &mockCompFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return nil, assert.AnError
		},
	}

	a := NewComponentRegistry(fetcher)
	_, err := a.ListComponents()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch registry.json")
}

func TestComponentRegistryAdapter_GetComponent_Success(t *testing.T) {
	t.Parallel()

	reg := registryFile{
		Components: map[string]registryComponentEntry{
			"mycomp": {
				Description: "My component",
				DistTags:    map[string]string{"latest": "1.0.0"},
				Versions: map[string]registryVersionEntry{
					"1.0.0": {Path: "mycomp/1.0.0.typ", Dependencies: nil},
				},
			},
		},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)

	fetcher := &mockCompFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return data, nil
		},
	}

	a := NewComponentRegistry(fetcher)
	info, err := a.GetComponent("mycomp")
	require.NoError(t, err)
	assert.Equal(t, "mycomp", info.Name)
	assert.Equal(t, "My component", info.Description)
	require.Contains(t, info.Versions, "1.0.0")
	assert.Equal(t, "mycomp/1.0.0.typ", info.Versions["1.0.0"].Path)
}

func TestComponentRegistryAdapter_GetComponent_NotFound(t *testing.T) {
	t.Parallel()

	reg := registryFile{
		Components: map[string]registryComponentEntry{},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)

	fetcher := &mockCompFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return data, nil
		},
	}

	a := NewComponentRegistry(fetcher)
	_, err = a.GetComponent("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestComponentRegistryAdapter_ResolveVersion_Success(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	v2, _ := semver.NewVersion("2.0.0")

	reg := registryFile{
		Components: map[string]registryComponentEntry{
			"mycomp": {
				Description: "My component",
				DistTags:    map[string]string{"latest": "2.0.0"},
				Versions: map[string]registryVersionEntry{
					"1.0.0": {Path: "mycomp/1.0.0.typ", Dependencies: nil},
					"2.0.0": {Path: "mycomp/2.0.0.typ", Dependencies: nil},
				},
			},
		},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)

	fetcher := &mockCompFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return data, nil
		},
	}

	a := NewComponentRegistry(fetcher)
	resolved, info, cv, err := a.ResolveVersion("mycomp", "latest")
	require.NoError(t, err)
	assert.True(t, resolved.Equal(v2))
	assert.Equal(t, "mycomp", info.Name)
	assert.Equal(t, "mycomp/2.0.0.typ", cv.Path)

	// Test caret range
	resolved, _, cv, err = a.ResolveVersion("mycomp", "^1.0.0")
	require.NoError(t, err)
	assert.True(t, resolved.Equal(v1))
	assert.Equal(t, "mycomp/1.0.0.typ", cv.Path)
}

func TestComponentRegistryAdapter_ResolveVersion_ComponentNotFound(t *testing.T) {
	t.Parallel()

	reg := registryFile{
		Components: map[string]registryComponentEntry{},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)

	fetcher := &mockCompFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return data, nil
		},
	}

	a := NewComponentRegistry(fetcher)
	_, _, _, err = a.ResolveVersion("nonexistent", "latest")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestComponentRegistryAdapter_ResolveVersion_NoMatchingVersion(t *testing.T) {
	t.Parallel()

	reg := registryFile{
		Components: map[string]registryComponentEntry{
			"mycomp": {
				Description: "My component",
				DistTags:    map[string]string{"latest": "1.0.0"},
				Versions: map[string]registryVersionEntry{
					"1.0.0": {Path: "mycomp/1.0.0.typ", Dependencies: nil},
				},
			},
		},
	}
	data, err := json.Marshal(reg)
	require.NoError(t, err)

	fetcher := &mockCompFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return data, nil
		},
	}

	a := NewComponentRegistry(fetcher)
	_, _, _, err = a.ResolveVersion("mycomp", ">=5.0.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no version matching")
}

func TestComponentRegistryAdapter_FetchComponentFile_Success(t *testing.T) {
	t.Parallel()

	fileContent := []byte("# My Component")
	fetcher := &mockCompFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			assert.Equal(t, "mycomp/1.0.0.typ", path)
			return fileContent, nil
		},
	}

	a := NewComponentRegistry(fetcher)
	info := ports.ComponentInfo{Name: "mycomp"}
	cv := &ports.ComponentVersion{Path: "mycomp/1.0.0.typ"}

	data, err := a.FetchComponentFile(info, cv)
	require.NoError(t, err)
	assert.Equal(t, fileContent, data)
}

func TestComponentRegistryAdapter_FetchComponentFile_Error(t *testing.T) {
	t.Parallel()

	fetcher := &mockCompFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return nil, assert.AnError
		},
	}

	a := NewComponentRegistry(fetcher)
	info := ports.ComponentInfo{Name: "mycomp"}
	cv := &ports.ComponentVersion{Path: "mycomp/1.0.0.typ"}

	_, err := a.FetchComponentFile(info, cv)
	require.Error(t, err)
}

func TestComponentRegistryAdapter_ConvertComponent(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")

	fetcher := &mockCompFetcher{}
	a := NewComponentRegistry(fetcher)

	entry := registryComponentEntry{
		Description: "Test component",
		DistTags:    map[string]string{"latest": "1.0.0"},
		Versions: map[string]registryVersionEntry{
			"1.0.0": {Path: "test/1.0.0.typ", Dependencies: []string{"dep@^1.0.0"}},
		},
	}

	info := a.convertComponent("test", entry)
	assert.Equal(t, "test", info.Name)
	assert.Equal(t, "Test component", info.Description)
	assert.Contains(t, info.DistTags, "latest")
	assert.True(t, info.DistTags["latest"].Equal(v1))
	require.Contains(t, info.Versions, "1.0.0")
	assert.Equal(t, "test/1.0.0.typ", info.Versions["1.0.0"].Path)
	assert.Equal(t, []string{"dep@^1.0.0"}, info.Versions["1.0.0"].Dependencies)
}

func TestComponentRegistryAdapter_ConvertComponent_InvalidVersion(t *testing.T) {
	t.Parallel()

	fetcher := &mockCompFetcher{}
	a := NewComponentRegistry(fetcher)

	entry := registryComponentEntry{
		Description: "Test",
		DistTags:    map[string]string{},
		Versions: map[string]registryVersionEntry{
			"not-a-version": {Path: "test/invalid.typ"},
		},
	}

	info := a.convertComponent("test", entry)
	assert.Empty(t, info.Versions, "invalid version should be skipped")
}

func TestComponentRegistryAdapter_WithHTTPServer(t *testing.T) {
	t.Parallel()

	reg := registryFile{
		Components: map[string]registryComponentEntry{
			"server-comp": {
				Description: "Server component",
				DistTags:    map[string]string{"latest": "1.0.0"},
				Versions: map[string]registryVersionEntry{
					"1.0.0": {Path: "server-comp/1.0.0.typ", Dependencies: nil},
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

	fetcher := &mockCompFetcher{
		rawFunc: func(repo, ref, path string) ([]byte, error) {
			return fetchHTTP(srv.URL)
		},
	}

	a := NewComponentRegistry(fetcher)
	components, err := a.ListComponents()
	require.NoError(t, err)
	require.Len(t, components, 1)
	assert.Equal(t, "server-comp", components[0].Name)
}
