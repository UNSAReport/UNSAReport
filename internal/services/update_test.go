package services

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/UNSAReport/UNSAReport/internal/mocks"
	"github.com/UNSAReport/UNSAReport/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUpdateService_SyncComponents_EmptyManifest(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	svc := NewUpdateService(
		WithUpdateComponentService(&ComponentService{}),
		WithUpdateStdout(&stdout),
		WithUpdateStderr(&bytes.Buffer{}),
	)

	err := svc.syncComponents(context.Background(), map[string]string{}, ports.UnsareportConfig{})
	require.NoError(t, err)
	assert.Empty(t, stdout.String())
}

func TestUpdateService_SyncComponents_NilManifest(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	svc := NewUpdateService(
		WithUpdateComponentService(&ComponentService{}),
		WithUpdateStdout(&stdout),
		WithUpdateStderr(&bytes.Buffer{}),
	)

	err := svc.syncComponents(context.Background(), nil, ports.UnsareportConfig{})
	require.NoError(t, err)
	assert.Empty(t, stdout.String())
}

func TestUpdateService_SyncComponents_InstallMissing(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	info := &ports.ComponentInfo{Name: "newcomp"}
	cv := &ports.ComponentVersion{Path: "newcomp/1.0.0.typ"}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "newcomp", "^1.0.0").Return(v1, info, cv, nil)
	reg.On("FetchComponentFile", *info, cv).Return([]byte("# newcomp"), nil)
	reg.On("ResolveVersion", mock.Anything, mock.Anything).Return(v1, info, cv, nil).Maybe()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)
	fs.On("FileExists", mock.Anything).Return(false)
	fs.On("EnsureDir", mock.Anything).Return(nil)
	fs.On("WriteFileAtomic", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{
			Components: map[string]ports.ComponentConfigEntry{},
		}, true, nil)
	cfg.On("ReadLockfile", "/project").Return(ports.Lockfile{
		Packages: map[string]ports.LockfilePackage{},
	}, nil)
	cfg.On("WriteLockfile", "/project", mock.Anything).Return(nil)
	cfg.On("WriteConfig", "/project", mock.Anything).Return(nil)

	compSvc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	var stdout bytes.Buffer
	svc := NewUpdateService(
		WithUpdateComponentService(compSvc),
		WithUpdateStdout(&stdout),
		WithUpdateStderr(&bytes.Buffer{}),
	)

	err := svc.syncComponents(context.Background(), map[string]string{
		"newcomp": "^1.0.0",
	}, ports.UnsareportConfig{})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Installing newcomp")
}

func TestUpdateService_SyncComponents_SkipSatisfied(t *testing.T) {
	t.Parallel()

	compSvc := NewComponentService(
		WithComponentRegistry(mocks.NewComponentRegistry(t)),
		WithComponentFS(mocks.NewFileSystem(t)),
		WithComponentConfig(mocks.NewConfigStore(t)),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	var stdout bytes.Buffer
	svc := NewUpdateService(
		WithUpdateComponentService(compSvc),
		WithUpdateStdout(&stdout),
		WithUpdateStderr(&bytes.Buffer{}),
	)

	err := svc.syncComponents(context.Background(), map[string]string{
		"mycomp": "^1.0.0",
	}, ports.UnsareportConfig{
		Components: map[string]ports.ComponentConfigEntry{
			"mycomp": {Version: "1.5.0"},
		},
	})
	require.NoError(t, err)
	assert.NotContains(t, stdout.String(), "Installing")
	assert.NotContains(t, stdout.String(), "Updating")
}

func TestUpdateService_SyncComponents_UpdateMismatched(t *testing.T) {
	t.Parallel()

	v2, _ := semver.NewVersion("2.0.0")
	info := &ports.ComponentInfo{Name: "mycomp"}
	cv := &ports.ComponentVersion{Path: "mycomp/2.0.0.typ"}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "mycomp", "^2.0.0").Return(v2, info, cv, nil)
	reg.On("FetchComponentFile", *info, cv).Return([]byte("# mycomp v2"), nil)
	reg.On("ResolveVersion", mock.Anything, mock.Anything).Return(v2, info, cv, nil).Maybe()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)
	fs.On("FileExists", mock.Anything).Return(false)
	fs.On("EnsureDir", mock.Anything).Return(nil)
	fs.On("WriteFileAtomic", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{
			Components: map[string]ports.ComponentConfigEntry{
				"mycomp": {Version: "1.0.0"},
			},
		}, true, nil)
	cfg.On("ReadLockfile", "/project").Return(ports.Lockfile{
		Packages: map[string]ports.LockfilePackage{},
	}, nil)
	cfg.On("WriteLockfile", "/project", mock.Anything).Return(nil)
	cfg.On("WriteConfig", "/project", mock.Anything).Return(nil)

	compSvc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	var stdout bytes.Buffer
	svc := NewUpdateService(
		WithUpdateComponentService(compSvc),
		WithUpdateStdout(&stdout),
		WithUpdateStderr(&bytes.Buffer{}),
	)

	err := svc.syncComponents(context.Background(), map[string]string{
		"mycomp": "^2.0.0",
	}, ports.UnsareportConfig{
		Components: map[string]ports.ComponentConfigEntry{
			"mycomp": {Version: "1.0.0"},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Updating mycomp")
}

func TestUpdateService_SyncComponents_NoteExtra(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	info := &ports.ComponentInfo{Name: "other"}
	cv := &ports.ComponentVersion{Path: "other/1.0.0.typ"}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "other", "^1.0.0").Return(v1, info, cv, nil)
	reg.On("FetchComponentFile", *info, cv).Return([]byte("# other"), nil)

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)
	fs.On("FileExists", mock.Anything).Return(false)
	fs.On("EnsureDir", mock.Anything).Return(nil)
	fs.On("WriteFileAtomic", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{
			Components: map[string]ports.ComponentConfigEntry{},
		}, true, nil)
	cfg.On("ReadLockfile", "/project").Return(ports.Lockfile{
		Packages: map[string]ports.LockfilePackage{},
	}, nil)
	cfg.On("WriteLockfile", "/project", mock.Anything).Return(nil)
	cfg.On("WriteConfig", "/project", mock.Anything).Return(nil)

	compSvc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	var stdout bytes.Buffer
	svc := NewUpdateService(
		WithUpdateComponentService(compSvc),
		WithUpdateStdout(&stdout),
		WithUpdateStderr(&bytes.Buffer{}),
	)

	err := svc.syncComponents(context.Background(), map[string]string{
		"other": "^1.0.0",
	}, ports.UnsareportConfig{
		Components: map[string]ports.ComponentConfigEntry{
			"extra": {Version: "1.0.0"},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Note: Component extra is installed but not required")
}

func TestUpdateService_SyncComponents_StdoutWriteError(t *testing.T) {
	t.Parallel()

	compSvc := NewComponentService(
		WithComponentRegistry(mocks.NewComponentRegistry(t)),
		WithComponentFS(mocks.NewFileSystem(t)),
		WithComponentConfig(mocks.NewConfigStore(t)),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	stdout := &errWriter{err: fmt.Errorf("write failed")}
	svc := NewUpdateService(
		WithUpdateComponentService(compSvc),
		WithUpdateStdout(stdout),
		WithUpdateStderr(&bytes.Buffer{}),
	)

	err := svc.syncComponents(context.Background(), map[string]string{
		"comp": "^1.0.0",
	}, ports.UnsareportConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write message")
}

func TestUpdateService_BuildUpdateEntries_SingleMode(t *testing.T) {
	t.Parallel()

	m := &Manifest{
		Mode: "single",
		Entries: []Entry{
			{Kind: KindFile, Src: "src/report.typ", Dest: "report.typ", Updatable: true},
			{Kind: KindFile, Src: "src/lib.typ", Dest: "lib.typ", Updatable: true},
		},
	}

	cfg := ports.UnsareportConfig{
		Mode: "single",
		Prepare: ports.PrepareConfig{
			Input: ports.PrepareInputConfig{
				SrcDir:     "source",
				ReportFile: "main.typ",
			},
			Output: ports.PrepareOutputConfig{
				SubmissionDir: "out",
			},
		},
	}

	svc := NewUpdateService()
	entries := svc.buildUpdateEntries(m, false, cfg, "")

	require.Len(t, entries, 2)
	destMap := make(map[string]Entry)
	for _, e := range entries {
		destMap[e.Dest] = e
	}
	assert.Contains(t, destMap, "main.typ")
	assert.Contains(t, destMap, "lib.typ")
}

func TestUpdateService_BuildUpdateEntries_MultiMode(t *testing.T) {
	t.Parallel()

	m := &Manifest{
		Mode: "multi",
		Entries: MultiEntrySet{
			Root: []Entry{
				{Kind: KindFile, Src: "src/report.typ", Dest: "report.typ", Updatable: true},
			},
			LabFiles: []Entry{
				{Kind: KindFile, Src: "src/lab.typ", Dest: "labs/{lab}/lab.typ", Updatable: true},
			},
		},
	}

	cfg := ports.UnsareportConfig{
		Mode:     "multi",
		Sessions: []string{"lab1", "lab2"},
		Prepare: ports.PrepareConfig{
			Input: ports.PrepareInputConfig{
				SrcDir:     "source",
				ReportFile: "main.typ",
			},
			Output: ports.PrepareOutputConfig{
				SubmissionDir: "out",
			},
		},
	}

	svc := NewUpdateService()
	entries := svc.buildUpdateEntries(m, true, cfg, "")

	dests := make(map[string]bool)
	for _, e := range entries {
		dests[e.Dest] = true
	}
	assert.True(t, dests["main.typ"])
	assert.True(t, dests["labs/lab1/lab.typ"])
	assert.True(t, dests["labs/lab2/lab.typ"])
}

func TestUpdateService_BuildUpdateEntries_MultiMode_SpecificSession(t *testing.T) {
	t.Parallel()

	m := &Manifest{
		Mode: "multi",
		Entries: MultiEntrySet{
			Root: []Entry{
				{Kind: KindFile, Src: "src/report.typ", Dest: "report.typ", Updatable: true},
			},
			LabFiles: []Entry{
				{Kind: KindFile, Src: "src/lab.typ", Dest: "labs/{lab}/lab.typ", Updatable: true},
			},
		},
	}

	cfg := ports.UnsareportConfig{
		Mode:     "multi",
		Sessions: []string{"lab1", "lab2"},
		Prepare: ports.PrepareConfig{
			Input: ports.PrepareInputConfig{
				SrcDir:     "source",
				ReportFile: "main.typ",
			},
			Output: ports.PrepareOutputConfig{
				SubmissionDir: "out",
			},
		},
	}

	svc := NewUpdateService()
	entries := svc.buildUpdateEntries(m, true, cfg, "lab3")

	dests := make(map[string]bool)
	for _, e := range entries {
		dests[e.Dest] = true
	}
	assert.True(t, dests["main.typ"])
	assert.True(t, dests["labs/lab3/lab.typ"])
	assert.False(t, dests["labs/lab1/lab.typ"])
	assert.False(t, dests["labs/lab2/lab.typ"])
}

func TestUpdateService_BuildUpdateEntries_SingleMode_Dedup(t *testing.T) {
	t.Parallel()

	m := &Manifest{
		Mode: "single",
		Entries: []Entry{
			{Kind: KindFile, Src: "src/a.typ", Dest: "dup.typ", Updatable: true},
			{Kind: KindFile, Src: "src/b.typ", Dest: "dup.typ", Updatable: true},
		},
	}

	cfg := ports.UnsareportConfig{
		Mode: "single",
		Prepare: ports.PrepareConfig{
			Input: ports.PrepareInputConfig{
				SrcDir:     "source",
				ReportFile: "main.typ",
			},
		},
	}

	svc := NewUpdateService()
	entries := svc.buildUpdateEntries(m, false, cfg, "")

	dupCount := 0
	for _, e := range entries {
		if e.Dest == "dup.typ" {
			dupCount++
		}
	}
	assert.Equal(t, 1, dupCount, "duplicate entries should be deduplicated")
}

func TestUpdateService_BuildUpdateEntries_SingleMode_InvalidEntries(t *testing.T) {
	t.Parallel()

	m := &Manifest{
		Mode:    "single",
		Entries: "not-an-array",
	}

	svc := NewUpdateService()
	entries := svc.buildUpdateEntries(m, false, ports.UnsareportConfig{}, "")

	assert.Nil(t, entries)
}

func TestUpdateService_BuildUpdateEntries_MultiMode_InvalidEntries(t *testing.T) {
	t.Parallel()

	m := &Manifest{
		Mode:    "multi",
		Entries: "not-a-multi-entry-set",
	}

	svc := NewUpdateService()
	entries := svc.buildUpdateEntries(m, true, ports.UnsareportConfig{}, "")

	assert.Nil(t, entries)
}

func TestUpdateService_SyncComponents_InvalidConstraint(t *testing.T) {
	t.Parallel()

	compSvc := NewComponentService(
		WithComponentRegistry(mocks.NewComponentRegistry(t)),
		WithComponentFS(mocks.NewFileSystem(t)),
		WithComponentConfig(mocks.NewConfigStore(t)),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	var stdout bytes.Buffer
	svc := NewUpdateService(
		WithUpdateComponentService(compSvc),
		WithUpdateStdout(&stdout),
		WithUpdateStderr(&bytes.Buffer{}),
	)

	err := svc.syncComponents(context.Background(), map[string]string{
		"mycomp": "not-a-valid-semver-constraint!!!",
	}, ports.UnsareportConfig{
		Components: map[string]ports.ComponentConfigEntry{
			"mycomp": {Version: "1.0.0"},
		},
	})
	require.NoError(t, err)
}

type errWriter struct {
	err error
}

func (w *errWriter) Write(p []byte) (int, error) {
	return 0, w.err
}
