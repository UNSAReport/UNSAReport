package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"github.com/UNSAReport/UNSAReport/internal/adapters/osfs"
	"github.com/UNSAReport/UNSAReport/internal/mocks"
	"github.com/UNSAReport/UNSAReport/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInstallService_Execute_MissingManifest(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("EnsureDir", mock.Anything).Return(nil).Maybe()
	fs.On("FileExists", mock.Anything).Return(false).Maybe()
	fs.On("WriteFileAtomic", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", mock.Anything).Return("/tmp/test-install", ports.UnsareportConfig{}, false, nil)
	cfg.On("WriteConfig", mock.Anything, mock.Anything).Return(nil).Maybe()
	cfg.On("WriteLockfile", mock.Anything, mock.Anything).Return(nil).Maybe()

	fetcher := mocks.NewTemplateFetcher(t)
	fetcher.On("Fetch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(map[string][]byte{"report.typ": []byte("content")}, nil)

	reg := mocks.NewTemplateRegistry(t)
	reg.On("GetTemplateVersion", "lab", "latest").
		Return(ports.TemplateInfo{Name: "lab", Version: "1.0.0", Path: "templates/lab"}, nil)

	var stdout bytes.Buffer
	svc := NewInstallService(
		WithInstallFetcher(fetcher),
		WithInstallFS(fs),
		WithInstallConfig(cfg),
		WithInstallRegistry(reg),
		WithInstallStdout(&stdout),
		WithInstallStderr(io.Discard),
	)

	err := svc.Execute(context.Background(), InstallOptions{
		Dest:     "/tmp/test-install",
		Template: "lab",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest.json not found")
}

func TestInstallService_Execute_InvalidManifest(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("EnsureDir", mock.Anything).Return(nil).Maybe()

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", mock.Anything).Return("/tmp/test-install", ports.UnsareportConfig{}, false, nil)

	fetcher := mocks.NewTemplateFetcher(t)
	fetcher.On("Fetch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(map[string][]byte{
			"manifest.json": []byte(`{"mode": "invalid_mode"}`),
		}, nil)

	reg := mocks.NewTemplateRegistry(t)
	reg.On("GetTemplateVersion", "lab", "latest").
		Return(ports.TemplateInfo{Name: "lab", Version: "1.0.0", Path: "templates/lab"}, nil)

	var stdout bytes.Buffer
	svc := NewInstallService(
		WithInstallFetcher(fetcher),
		WithInstallFS(fs),
		WithInstallConfig(cfg),
		WithInstallRegistry(reg),
		WithInstallStdout(&stdout),
		WithInstallStderr(io.Discard),
	)

	err := svc.Execute(context.Background(), InstallOptions{
		Dest:     "/tmp/test-install",
		Template: "lab",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load manifest")
}

func TestInstallService_Execute_SessionWithNonMultiMode(t *testing.T) {
	t.Parallel()

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", mock.Anything).
		Return("/tmp/test", ports.UnsareportConfig{Mode: "single"}, true, nil)

	var stdout bytes.Buffer
	svc := NewInstallService(
		WithInstallConfig(cfg),
		WithInstallStdout(&stdout),
		WithInstallStderr(io.Discard),
	)

	err := svc.Execute(context.Background(), InstallOptions{
		Dest:     "/tmp/test",
		Template: "lab",
		Session:  "l1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--session flag can only be used with multi-mode")
}

func TestInstallService_Execute_RegistryError(t *testing.T) {
	t.Parallel()

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", mock.Anything).Return("/tmp/test", ports.UnsareportConfig{}, false, nil)

	reg := mocks.NewTemplateRegistry(t)
	reg.On("GetTemplateVersion", mock.Anything, mock.Anything).
		Return(ports.TemplateInfo{}, fmt.Errorf("template not found"))

	var stdout bytes.Buffer
	svc := NewInstallService(
		WithInstallConfig(cfg),
		WithInstallRegistry(reg),
		WithInstallStdout(&stdout),
		WithInstallStderr(io.Discard),
	)

	err := svc.Execute(context.Background(), InstallOptions{
		Dest:     "/tmp/test",
		Template: "nonexistent",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get template")
}

func TestInstallService_Execute_FetcherError(t *testing.T) {
	t.Parallel()

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", mock.Anything).Return("/tmp/test", ports.UnsareportConfig{}, false, nil)

	reg := mocks.NewTemplateRegistry(t)
	reg.On("GetTemplateVersion", "lab", "latest").
		Return(ports.TemplateInfo{Name: "lab", Version: "1.0.0", Path: "templates/lab"}, nil)

	fetcher := mocks.NewTemplateFetcher(t)
	fetcher.On("Fetch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("network error"))

	var stdout bytes.Buffer
	svc := NewInstallService(
		WithInstallFetcher(fetcher),
		WithInstallConfig(cfg),
		WithInstallRegistry(reg),
		WithInstallStdout(&stdout),
		WithInstallStderr(io.Discard),
	)

	err := svc.Execute(context.Background(), InstallOptions{
		Dest:     "/tmp/test",
		Template: "lab",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch templates")
}

func TestParseTemplateArg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		arg       string
		wantName  string
		wantRange string
	}{
		{"name only", "lab", "lab", "latest"},
		{"name with version", "lab@1.0.0", "lab", "1.0.0"},
		{"name with caret range", "lab@^1.0.0", "lab", "^1.0.0"},
		{"name with tilde range", "lab@~1.2.0", "lab", "~1.2.0"},
		{"empty string", "", "", "latest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotName, gotRange := parseTemplateArg(tt.arg)
			assert.Equal(t, tt.wantName, gotName)
			assert.Equal(t, tt.wantRange, gotRange)
		})
	}
}

func TestSubstituteLab_Install(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entries  []Entry
		lab      string
		expected []string
	}{
		{
			name: "single entry",
			entries: []Entry{
				{Kind: KindFile, Src: "report.typ", Dest: "{lab}/report.typ"},
			},
			lab:      "l1",
			expected: []string{"l1/report.typ"},
		},
		{
			name: "multiple entries",
			entries: []Entry{
				{Kind: KindFile, Src: "report.typ", Dest: "{lab}/report.typ"},
				{Kind: KindFile, Src: "main.typ", Dest: "{lab}/main.typ"},
			},
			lab:      "l3",
			expected: []string{"l3/report.typ", "l3/main.typ"},
		},
		{
			name: "no lab placeholder",
			entries: []Entry{
				{Kind: KindFile, Src: "bib.bib", Dest: "common/bib.bib"},
			},
			lab:      "l1",
			expected: []string{"common/bib.bib"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := substituteLab(tt.entries, tt.lab)
			require.Len(t, result, len(tt.expected))
			for i, e := range result {
				assert.Equal(t, tt.expected[i], e.Dest)
			}
		})
	}
}

func TestInstallService_Execute_Success(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	manifestJSON := []byte(`{
		"mode": "single",
		"entries": [
			{"kind": "file", "src": "report.typ", "dest": "report.typ"},
			{"kind": "dir", "src": "images", "dest": "images"},
			{"kind": "file", "src": "images/logo.png", "dest": "images/logo.png"}
		]
	}`)
	templateFiles := map[string][]byte{
		"manifest.json":   manifestJSON,
		"report.typ":      []byte("#set page(paper: \"a4\")\n= Lab"),
		"images/logo.png": []byte("fake png"),
	}

	fetcher := mocks.NewTemplateFetcher(t)
	fetcher.On("Fetch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(templateFiles, nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", mock.Anything).Return(tmpDir, ports.UnsareportConfig{}, false, nil)
	cfg.On("WriteConfig", mock.Anything, mock.Anything).Return(nil)
	cfg.On("WriteLockfile", mock.Anything, mock.Anything).Return(nil)
	cfg.On("ReadLockfile", mock.Anything).Return(ports.Lockfile{Packages: make(map[string]ports.LockfilePackage)}, nil)

	reg := mocks.NewTemplateRegistry(t)
	reg.On("GetTemplateVersion", "lab", "latest").
		Return(ports.TemplateInfo{Name: "lab", Version: "1.0.0", Path: "templates/lab"}, nil)

	// Use the real osfs adapter instead of a copy-pasted mock
	realFS := osfs.New()
	var stdout bytes.Buffer
	svc := NewInstallService(
		WithInstallFetcher(fetcher),
		WithInstallFS(realFS),
		WithInstallConfig(cfg),
		WithInstallRegistry(reg),
		WithInstallStdout(&stdout),
		WithInstallStderr(io.Discard),
	)

	err := svc.Execute(context.Background(), InstallOptions{
		Dest:     tmpDir,
		Template: "lab",
	})
	require.NoError(t, err)

	// Verify files were actually written to the real filesystem
	assert.FileExists(t, filepath.Join(tmpDir, "report.typ"))
	assert.DirExists(t, filepath.Join(tmpDir, "images"))
	assert.FileExists(t, filepath.Join(tmpDir, "images", "logo.png"))
}
