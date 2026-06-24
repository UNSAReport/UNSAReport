//go:build integration

package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/UNSAReport/UNSAReport/internal/adapters/config"
	"github.com/UNSAReport/UNSAReport/internal/adapters/osfs"
	"github.com/UNSAReport/UNSAReport/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTemplateFiles returns the minimal set of files needed to simulate a single-mode template.
func mockTemplateFiles() map[string][]byte {
	return map[string][]byte{
		"manifest.json": []byte(`{
			"mode": "single",
			"entries": [
				{"kind": "file", "src": "report.typ", "dest": "report.typ"},
				{"kind": "dir", "src": "images", "dest": "images"},
				{"kind": "file", "src": "images/logo.png", "dest": "images/logo.png"},
				{"kind": "file", "src": "bibliography.bib", "dest": "bibliography.bib"}
			]
		}`),
		"report.typ":       []byte(`#set page(paper: "a4")\n= Lab Report`),
		"images/logo.png":  {0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
		"bibliography.bib": []byte(`@article{test, title={Test}}`),
	}
}

// mockMultiTemplateFiles returns files for a multi-lab template.
func mockMultiTemplateFiles() map[string][]byte {
	return map[string][]byte{
		"manifest.json": []byte(`{
			"mode": "multi",
			"entries": {
				"root": [
					{"kind": "file", "src": "common/bibliography.bib", "dest": "common/bibliography.bib"}
				],
				"labFiles": [
					{"kind": "file", "src": "report.typ", "dest": "{lab}/report.typ"}
				]
			}
		}`),
		"common/bibliography.bib": []byte(`@article{test, title={Test}}`),
		"report.typ":             []byte(`#set page(paper: "a4")\n= Lab Report`),
	}
}

func TestE2E_SingleModeWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	realFS := osfs.New()
	realCfg := config.New()

	// --- Phase 1: Install ---
	t.Run("install", func(t *testing.T) {
		manifestData := mockTemplateFiles()
		fetcher := &e2eMockFetch{files: manifestData}
		reg := &e2eMockRegistry{
			templates: []ports.TemplateInfo{
				{Name: "lab", Description: "Single lab template", Version: "1.0.0", Path: "templates/lab"},
			},
		}

		var stdout bytes.Buffer
		svc := NewInstallService(
			WithInstallFetcher(fetcher),
			WithInstallFS(realFS),
			WithInstallConfig(realCfg),
			WithInstallRegistry(reg),
			WithInstallStdout(&stdout),
			WithInstallStderr(io.Discard),
		)

		err := svc.Execute(ctx, InstallOptions{
			Dest:     tmpDir,
			Template: "lab",
		})
		require.NoError(t, err, "install should succeed")

		// Verify config was written
		cfgPath := filepath.Join(tmpDir, "unsareport.json")
		data, err := os.ReadFile(cfgPath)
		require.NoError(t, err, "unsareport.json should exist")

		var cfgWritten ports.UnsareportConfig
		require.NoError(t, json.Unmarshal(data, &cfgWritten))
		assert.Equal(t, "lab", cfgWritten.Template)
		assert.Equal(t, "1.0.0", cfgWritten.TemplateVersion)
		assert.Equal(t, "single", cfgWritten.Mode)

		// Verify template files were written
		reportPath := filepath.Join(tmpDir, "report.typ")
		assert.FileExists(t, reportPath)
		reportData, err := os.ReadFile(reportPath)
		require.NoError(t, err)
		assert.Contains(t, string(reportData), "Lab Report")

		assert.DirExists(t, filepath.Join(tmpDir, "images"))
		assert.FileExists(t, filepath.Join(tmpDir, "images", "logo.png"))
		assert.FileExists(t, filepath.Join(tmpDir, "bibliography.bib"))

		assert.Contains(t, stdout.String(), "Installation complete!")
	})

	// --- Phase 2: Capture ---
	t.Run("capture", func(t *testing.T) {
		var renderedPath string
		var renderedCommands []ports.CaptureCommand
		renderer := &e2eMockRenderer{
			renderFn: func(ctx context.Context, resultPath string, commands []ports.CaptureCommand, flags []string, cfg ports.CaptureConfig) (string, error) {
				renderedPath = resultPath
				renderedCommands = commands
				return "mock output", nil
			},
		}

		var stdout bytes.Buffer
		svc := NewCaptureService(
			WithCaptureRenderer(renderer),
			WithCaptureFS(realFS),
			WithCaptureConfig(realCfg),
			WithCaptureStdout(&stdout),
			WithCaptureStderr(io.Discard),
		)

		// We need to chdir to tmpDir for the capture to find the config
		origDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(origDir)

		err = svc.Execute(ctx, CaptureOptions{
			Args: []string{"output.png", "ls -la", "cat README.md"},
		})
		require.NoError(t, err, "capture should succeed")

		assert.Equal(t, "output.png", renderedPath)

		// 2 commands + 2 sleeps + 1 final sleep = 5
		require.Len(t, renderedCommands, 5)
		assert.Equal(t, "Command", renderedCommands[0].Type)
		assert.Equal(t, "ls -la", renderedCommands[0].Args)
		assert.Equal(t, "Sleep", renderedCommands[1].Type)
		assert.Equal(t, "Command", renderedCommands[2].Type)
		assert.Equal(t, "cat README.md", renderedCommands[2].Args)

		assert.Contains(t, stdout.String(), "Capturing instruction: ls -la")
	})

	// --- Phase 3: Prepare ---
	t.Run("prepare", func(t *testing.T) {
		compiler := &e2eMockCompiler{
			queryVarsFn: func(ctx context.Context, reportPath string) (map[string]string, error) {
				return map[string]string{
					"lab_number": "1",
					"course":     "SO",
					"members":    "John;Jane",
				}, nil
			},
			compileFn: func(ctx context.Context, reportPath, reportPDF string, inputs map[string]string) error {
				return os.WriteFile(reportPDF, []byte("compiled pdf"), 0o644)
			},
		}

		archiver := &e2eMockArchiver{
			archiveDirFn: func(zipPath, srcDir string) error {
				return createFakeZip(zipPath)
			},
			archiveFilesFn: func(zipPath, baseDir string, files []string) error {
				return createFakeZip(zipPath)
			},
		}

		// Create src directory
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "src"), 0o755))

		var stdout bytes.Buffer
		svc := NewPrepareService(
			WithPrepareCompiler(compiler),
			WithPrepareArchiver(archiver),
			WithPrepareFS(realFS),
			WithPrepareConfig(realCfg),
			WithPrepareStdout(&stdout),
			WithPrepareStderr(io.Discard),
		)

		origDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(origDir)

		err = svc.Execute(ctx, PrepareOptions{Configure: false}, "")
		require.NoError(t, err, "prepare should succeed")

		submissionDir := filepath.Join(tmpDir, "submission")
		assert.DirExists(t, submissionDir)
		assert.FileExists(t, filepath.Join(submissionDir, "Informe_1.pdf"))
		assert.FileExists(t, filepath.Join(submissionDir, "Código Fuente_1.zip"))

		assert.Contains(t, stdout.String(), "Compiling typst report...")
	})

	// --- Phase 4: Verify full output tree ---
	t.Run("verify output tree", func(t *testing.T) {
		submissionDir := filepath.Join(tmpDir, "submission")
		entries, err := os.ReadDir(submissionDir)
		require.NoError(t, err)

		files := make(map[string]bool)
		for _, e := range entries {
			files[e.Name()] = true
		}

		assert.True(t, files["Informe_1.pdf"], "should have Informe_1.pdf")
		assert.True(t, files["Código Fuente_1.zip"], "should have Código Fuente_1.zip")
	})
}

func TestE2E_MultiModeWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	realFS := osfs.New()
	realCfg := config.New()

	manifestData := mockMultiTemplateFiles()
	fetcher := &e2eMockFetch{files: manifestData}
	reg := &e2eMockRegistry{
		templates: []ports.TemplateInfo{
			{Name: "multi-lab", Description: "Multi lab template", Version: "1.0.0", Path: "templates/multi-lab"},
		},
	}

	// --- Phase 1: Install multi-mode template ---
	t.Run("install", func(t *testing.T) {
		var stdout bytes.Buffer
		svc := NewInstallService(
			WithInstallFetcher(fetcher),
			WithInstallFS(realFS),
			WithInstallConfig(realCfg),
			WithInstallRegistry(reg),
			WithInstallStdout(&stdout),
			WithInstallStderr(io.Discard),
		)

		err := svc.Execute(ctx, InstallOptions{
			Dest:     tmpDir,
			Template: "multi-lab",
		})
		require.NoError(t, err, "install should succeed")

		cfgPath := filepath.Join(tmpDir, "unsareport.json")
		data, err := os.ReadFile(cfgPath)
		require.NoError(t, err)

		var cfgWritten ports.UnsareportConfig
		require.NoError(t, json.Unmarshal(data, &cfgWritten))
		assert.Equal(t, "multi-lab", cfgWritten.Template)
		assert.Equal(t, "multi", cfgWritten.Mode)
		assert.Contains(t, cfgWritten.Sessions, "l1")

		assert.FileExists(t, filepath.Join(tmpDir, "common", "bibliography.bib"))
		assert.FileExists(t, filepath.Join(tmpDir, "l1", "report.typ"))
	})

	// --- Phase 2: Prepare in multi-mode ---
	t.Run("prepare", func(t *testing.T) {
		compiler := &e2eMockCompiler{
			queryVarsFn: func(ctx context.Context, reportPath string) (map[string]string, error) {
				return map[string]string{"lab_number": "1"}, nil
			},
			compileFn: func(ctx context.Context, reportPath, reportPDF string, inputs map[string]string) error {
				return os.WriteFile(reportPDF, []byte("compiled"), 0o644)
			},
		}

		archiver := &e2eMockArchiver{
			archiveDirFn: func(zipPath, srcDir string) error {
				return createFakeZip(zipPath)
			},
			archiveFilesFn: func(zipPath, baseDir string, files []string) error {
				return createFakeZip(zipPath)
			},
		}

		// Create l1/src directory
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "l1", "src"), 0o755))

		var stdout bytes.Buffer
		svc := NewPrepareService(
			WithPrepareCompiler(compiler),
			WithPrepareArchiver(archiver),
			WithPrepareFS(realFS),
			WithPrepareConfig(realCfg),
			WithPrepareStdout(&stdout),
			WithPrepareStderr(io.Discard),
		)

		// Run from project root with "l1" as lab dir argument
		origDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer os.Chdir(origDir)

		err = svc.Execute(ctx, PrepareOptions{Configure: false}, "l1")
		require.NoError(t, err, "prepare should succeed")

		submissionDir := filepath.Join(tmpDir, "l1", "submission")
		assert.DirExists(t, submissionDir)
		assert.FileExists(t, filepath.Join(submissionDir, "Informe_1.pdf"))
		assert.FileExists(t, filepath.Join(submissionDir, "Código Fuente_1.zip"))
	})
}

func TestE2E_InstallWithComponents(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	realFS := osfs.New()
	realCfg := config.New()

	manifestData := map[string][]byte{
		"manifest.json": []byte(`{
			"mode": "single",
			"components": {"code-block": "^1.0.0"},
			"entries": [
				{"kind": "file", "src": "report.typ", "dest": "report.typ"}
			]
		}`),
		"report.typ": []byte(`#import "@local/code-block:0.1.0": codeblock\n= Lab`),
	}

	fetcher := &e2eMockFetch{files: manifestData}
	reg := &e2eMockRegistry{
		templates: []ports.TemplateInfo{
			{Name: "lab", Version: "1.0.0", Path: "templates/lab"},
		},
	}

	compRegistry := &e2eMockCompRegistry{
		resolveFn: func(name, rangeSpec string) (*semver.Version, *ports.ComponentInfo, *ports.ComponentVersion, error) {
			v, _ := semver.NewVersion("1.0.0")
			info := &ports.ComponentInfo{
				Name:        name,
				Description: "Code block component",
				DistTags:    map[string]*semver.Version{"latest": v},
				Versions:    map[string]*ports.ComponentVersion{"1.0.0": {Version: v, Path: "components/" + name + "/typst.toml"}},
			}
			cv := &ports.ComponentVersion{
				Version: v,
				Path:    "components/" + name + "/typst.toml",
			}
			return v, info, cv, nil
		},
		fetchFn: func(info ports.ComponentInfo, cv *ports.ComponentVersion) ([]byte, error) {
			return []byte("// code-block component v1.0.0\n#let codeblock = none"), nil
		},
	}

	compSvc := NewComponentService(
		WithComponentFetcher(fetcher),
		WithComponentFS(realFS),
		WithComponentConfig(realCfg),
		WithComponentRegistry(compRegistry),
		WithComponentStdout(io.Discard),
		WithComponentStderr(io.Discard),
	)

	var stdout bytes.Buffer
	svc := NewInstallService(
		WithInstallFetcher(fetcher),
		WithInstallFS(realFS),
		WithInstallConfig(realCfg),
		WithInstallRegistry(reg),
		WithInstallComponentService(compSvc),
		WithInstallStdout(&stdout),
		WithInstallStderr(io.Discard),
	)

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(origDir)

	err = svc.Execute(ctx, InstallOptions{
		Dest:     tmpDir,
		Template: "lab",
	})
	require.NoError(t, err, "install with components should succeed")

	componentPath := filepath.Join(tmpDir, "components", "code-block.typ")
	assert.FileExists(t, componentPath, "component file should exist")
	componentData, err := os.ReadFile(componentPath)
	require.NoError(t, err)
	assert.Contains(t, string(componentData), "code-block component")
}

func TestE2E_UpdateWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	realFS := osfs.New()
	realCfg := config.New()

	// First install
	initialFiles := mockTemplateFiles()
	fetcher := &e2eMockFetch{files: initialFiles}
	reg := &e2eMockRegistry{
		templates: []ports.TemplateInfo{
			{Name: "lab", Version: "1.0.0", Path: "templates/lab"},
		},
	}

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(origDir)

	installSvc := NewInstallService(
		WithInstallFetcher(fetcher),
		WithInstallFS(realFS),
		WithInstallConfig(realCfg),
		WithInstallRegistry(reg),
		WithInstallStdout(io.Discard),
		WithInstallStderr(io.Discard),
	)
	require.NoError(t, installSvc.Execute(ctx, InstallOptions{Dest: tmpDir, Template: "lab"}))

	// Modify the local report.typ
	reportPath := filepath.Join(tmpDir, "report.typ")
	require.NoError(t, os.WriteFile(reportPath, []byte("# Lab Report - Local Edits"), 0o644))

	// Update with new remote version
	updatedFiles := mockTemplateFiles()
	updatedFiles["report.typ"] = []byte("#set page(paper: \"a4\")\n= Updated Lab Report")
	updatedFiles["manifest.json"] = []byte(`{
		"mode": "single",
		"entries": [
			{"kind": "file", "src": "report.typ", "dest": "report.typ", "updatable": true},
			{"kind": "dir", "src": "images", "dest": "images"},
			{"kind": "file", "src": "images/logo.png", "dest": "images/logo.png"},
			{"kind": "file", "src": "bibliography.bib", "dest": "bibliography.bib"}
		]
	}`)

	updatedReg := &e2eMockRegistry{
		templates: []ports.TemplateInfo{
			{Name: "lab", Version: "1.1.0", Path: "templates/lab"},
		},
	}
	updatedFetcher := &e2eMockFetch{files: updatedFiles}

	updateSvc := NewUpdateService(
		WithUpdateFetcher(updatedFetcher),
		WithUpdateFS(realFS),
		WithUpdateConfig(realCfg),
		WithUpdateRegistry(updatedReg),
		WithUpdateStdout(io.Discard),
		WithUpdateStderr(io.Discard),
	)

	err = updateSvc.Execute(ctx, UpdateOptions{
		Dest:  tmpDir,
		Force: true,
	})
	require.NoError(t, err, "update should succeed")

	reportData, err := os.ReadFile(reportPath)
	require.NoError(t, err)
	assert.Contains(t, string(reportData), "Updated Lab Report")

	cfgData, err := os.ReadFile(filepath.Join(tmpDir, "unsareport.json"))
	require.NoError(t, err)
	var updatedCfg ports.UnsareportConfig
	require.NoError(t, json.Unmarshal(cfgData, &updatedCfg))
	assert.Equal(t, "1.1.0", updatedCfg.TemplateVersion)
}

// createFakeZip creates a minimal valid ZIP file.
func createFakeZip(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	fw, err := w.Create("main.go")
	if err != nil {
		return err
	}
	_, err = fw.Write([]byte("package main\n"))
	if err != nil {
		return err
	}
	return w.Close()
}

// --- Mock implementations for E2E tests ---

type e2eMockFetch struct {
	files map[string][]byte
}

func (m *e2eMockFetch) Fetch(ctx context.Context, repo, ref, templatePath string) (map[string][]byte, error) {
	return m.files, nil
}

func (m *e2eMockFetch) FetchRaw(ctx context.Context, repo, ref, path string) ([]byte, error) {
	return nil, nil
}

func (m *e2eMockFetch) LoadLocal(dir string) (map[string][]byte, error) {
	return m.files, nil
}

type e2eMockRegistry struct {
	templates []ports.TemplateInfo
}

func (m *e2eMockRegistry) ListTemplates() ([]ports.TemplateInfo, error) {
	return m.templates, nil
}

func (m *e2eMockRegistry) GetTemplate(name string) (ports.TemplateInfo, error) {
	for _, t := range m.templates {
		if t.Name == name {
			return t, nil
		}
	}
	return ports.TemplateInfo{}, nil
}

func (m *e2eMockRegistry) GetTemplateVersion(name string, rangeSpec string) (ports.TemplateInfo, error) {
	for _, t := range m.templates {
		if t.Name == name {
			return t, nil
		}
	}
	if len(m.templates) > 0 {
		return m.templates[0], nil
	}
	return ports.TemplateInfo{}, nil
}

type e2eMockRenderer struct {
	renderFn func(ctx context.Context, resultPath string, commands []ports.CaptureCommand, flags []string, cfg ports.CaptureConfig) (string, error)
}

func (m *e2eMockRenderer) Render(ctx context.Context, resultPath string, commands []ports.CaptureCommand, flags []string, cfg ports.CaptureConfig) (string, error) {
	if m.renderFn != nil {
		return m.renderFn(ctx, resultPath, commands, flags, cfg)
	}
	return "", nil
}

type e2eMockCompiler struct {
	queryVarsFn func(ctx context.Context, reportPath string) (map[string]string, error)
	compileFn   func(ctx context.Context, reportPath, reportPDF string, inputs map[string]string) error
}

func (m *e2eMockCompiler) QueryVars(ctx context.Context, reportPath string) (map[string]string, error) {
	if m.queryVarsFn != nil {
		return m.queryVarsFn(ctx, reportPath)
	}
	return nil, nil
}

func (m *e2eMockCompiler) Compile(ctx context.Context, reportPath, reportPDF string, inputs map[string]string) error {
	if m.compileFn != nil {
		return m.compileFn(ctx, reportPath, reportPDF, inputs)
	}
	return nil
}

type e2eMockArchiver struct {
	archiveDirFn   func(zipPath, srcDir string) error
	archiveFilesFn func(zipPath, baseDir string, files []string) error
}

func (m *e2eMockArchiver) ArchiveDir(zipPath, srcDir string) error {
	if m.archiveDirFn != nil {
		return m.archiveDirFn(zipPath, srcDir)
	}
	return nil
}

func (m *e2eMockArchiver) ArchiveFiles(zipPath, baseDir string, files []string) error {
	if m.archiveFilesFn != nil {
		return m.archiveFilesFn(zipPath, baseDir, files)
	}
	return nil
}

type e2eMockCompRegistry struct {
	resolveFn func(name, rangeSpec string) (*semver.Version, *ports.ComponentInfo, *ports.ComponentVersion, error)
	fetchFn   func(info ports.ComponentInfo, cv *ports.ComponentVersion) ([]byte, error)
}

func (m *e2eMockCompRegistry) ListComponents() ([]ports.ComponentInfo, error) {
	return nil, nil
}

func (m *e2eMockCompRegistry) GetComponent(name string) (ports.ComponentInfo, error) {
	return ports.ComponentInfo{}, nil
}

func (m *e2eMockCompRegistry) ResolveVersion(name string, rangeSpec string) (*semver.Version, *ports.ComponentInfo, *ports.ComponentVersion, error) {
	if m.resolveFn != nil {
		return m.resolveFn(name, rangeSpec)
	}
	return nil, nil, nil, nil
}

func (m *e2eMockCompRegistry) FetchComponentFile(info ports.ComponentInfo, cv *ports.ComponentVersion) ([]byte, error) {
	if m.fetchFn != nil {
		return m.fetchFn(info, cv)
	}
	return nil, nil
}
