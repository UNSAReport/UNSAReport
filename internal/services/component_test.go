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

func TestParseComponentDep(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantName  string
		wantRange string
	}{
		{
			name:      "with version",
			input:     "foo@^1.0.0",
			wantName:  "foo",
			wantRange: "^1.0.0",
		},
		{
			name:      "without version",
			input:     "bar",
			wantName:  "bar",
			wantRange: "*",
		},
		{
			name:      "empty string",
			input:     "",
			wantName:  "",
			wantRange: "*",
		},
		{
			name:      "multiple at signs",
			input:     "foo@bar@baz",
			wantName:  "foo",
			wantRange: "bar@baz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotName, gotRange := parseComponentDep(tt.input)
			assert.Equal(t, tt.wantName, gotName)
			assert.Equal(t, tt.wantRange, gotRange)
		})
	}
}

func TestNewComponentService(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	cfg := mocks.NewConfigStore(t)
	reg := mocks.NewComponentRegistry(t)
	fetcher := mocks.NewTemplateFetcher(t)
	var stdout, stderr bytes.Buffer

	svc := NewComponentService(
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentRegistry(reg),
		WithComponentFetcher(fetcher),
		WithComponentStdout(&stdout),
		WithComponentStderr(&stderr),
	)

	require.NotNil(t, svc)
	assert.Equal(t, fs, svc.FS)
	assert.Equal(t, cfg, svc.Config)
	assert.Equal(t, reg, svc.Registry)
	assert.Equal(t, fetcher, svc.Fetcher)
	assert.Equal(t, &stdout, svc.Stdout)
	assert.Equal(t, &stderr, svc.Stderr)
}

func TestComponentService_Add_ResolveError(t *testing.T) {
	t.Parallel()

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "bad", "latest").
		Return(nil, nil, nil, fmt.Errorf("component not found"))

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(mocks.NewFileSystem(t)),
		WithComponentConfig(mocks.NewConfigStore(t)),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Add(context.Background(), "bad", "latest", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve version")
}

func TestComponentService_Add_Success(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	info := &ports.ComponentInfo{Name: "mycomp"}
	cv := &ports.ComponentVersion{Path: "mycomp/1.0.0.typ", Dependencies: nil}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "mycomp", "^1.0.0").Return(v1, info, cv, nil)
	reg.On("FetchComponentFile", *info, cv).Return([]byte("# mycomp"), nil)
	reg.On("ResolveVersion", mock.Anything, mock.Anything).Return(v1, info, cv, nil).Maybe()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)
	fs.On("FileExists", "/project/components/mycomp.typ").Return(false)
	fs.On("EnsureDir", "/project/components").Return(nil)
	fs.On("WriteFileAtomic", "/project/components/mycomp.typ", []byte("# mycomp"), mock.Anything).Return(nil)

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

	var stdout bytes.Buffer
	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&stdout),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Add(context.Background(), "mycomp", "^1.0.0", false)
	require.NoError(t, err)
}

func TestComponentService_Add_InvalidName(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	info := &ports.ComponentInfo{Name: "bad name!"}
	cv := &ports.ComponentVersion{Path: "bad/1.0.0.typ"}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "bad name!", "latest").Return(v1, info, cv, nil)

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{}, true, nil)

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Add(context.Background(), "bad name!", "latest", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid component name")
}

func TestComponentService_Add_NoProjectRoot(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	info := &ports.ComponentInfo{Name: "mycomp"}
	cv := &ports.ComponentVersion{Path: "mycomp/1.0.0.typ"}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "mycomp", "latest").Return(v1, info, cv, nil)

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/nowhere", nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/nowhere").
		Return("", ports.UnsareportConfig{}, false, nil)

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Add(context.Background(), "mycomp", "latest", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsareport.json not found")
}

func TestComponentService_Add_EmptyData(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	info := &ports.ComponentInfo{Name: "empty"}
	cv := &ports.ComponentVersion{Path: "empty/1.0.0.typ"}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "empty", "latest").Return(v1, info, cv, nil)
	reg.On("FetchComponentFile", *info, cv).Return([]byte{}, nil)

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)
	fs.On("FileExists", "/project/components/empty.typ").Return(false)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{}, true, nil)

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Add(context.Background(), "empty", "latest", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "returned empty data")
}

func TestComponentService_Add_FetchError(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	info := &ports.ComponentInfo{Name: "fail"}
	cv := &ports.ComponentVersion{Path: "fail/1.0.0.typ"}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "fail", "latest").Return(v1, info, cv, nil)
	reg.On("FetchComponentFile", *info, cv).Return(nil, fmt.Errorf("network error"))

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)
	fs.On("FileExists", "/project/components/fail.typ").Return(false)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{}, true, nil)

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Add(context.Background(), "fail", "latest", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch component")
}

func TestComponentService_Add_WithDeps(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	v2, _ := semver.NewVersion("2.0.0")
	info := &ports.ComponentInfo{Name: "parent"}
	cv := &ports.ComponentVersion{Path: "parent/1.0.0.typ", Dependencies: []string{"dep@^2.0.0"}}
	depInfo := &ports.ComponentInfo{Name: "dep"}
	depCv := &ports.ComponentVersion{Path: "dep/2.0.0.typ"}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "parent", "latest").Return(v1, info, cv, nil)
	reg.On("ResolveVersion", "dep", "^2.0.0").Return(v2, depInfo, depCv, nil)
	reg.On("FetchComponentFile", *info, cv).Return([]byte("# parent"), nil)

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

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Add(context.Background(), "parent", "latest", false)
	require.NoError(t, err)
}

func TestComponentService_Add_DepsUnsatisfiable(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	info := &ports.ComponentInfo{Name: "parent"}
	cv := &ports.ComponentVersion{Path: "parent/1.0.0.typ", Dependencies: []string{"missingdep@^1.0.0"}}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "parent", "latest").Return(v1, info, cv, nil)
	reg.On("ResolveVersion", "missingdep", "^1.0.0").
		Return(nil, nil, nil, fmt.Errorf("component not found"))
	reg.On("FetchComponentFile", *info, cv).Return([]byte("# parent"), nil)

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)
	fs.On("FileExists", mock.Anything).Return(false)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{}, true, nil)

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Add(context.Background(), "parent", "latest", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dependency")
}

func TestComponentService_Remove_Success(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)
	fs.On("FileExists", "/project/components/mycomp.typ").Return(true)
	fs.On("Remove", "/project/components/mycomp.typ").Return(nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{
			Components: map[string]ports.ComponentConfigEntry{
				"mycomp": {Version: "1.0.0"},
			},
		}, true, nil)
	cfg.On("ReadLockfile", "/project").Return(ports.Lockfile{
		Packages: map[string]ports.LockfilePackage{
			"components/mycomp.typ": {Version: "1.0.0"},
		},
	}, nil)
	cfg.On("WriteLockfile", "/project", mock.Anything).Return(nil)
	cfg.On("WriteConfig", "/project", mock.Anything).Return(nil)

	svc := NewComponentService(
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Remove(context.Background(), "mycomp")
	require.NoError(t, err)
}

func TestComponentService_Remove_NoProjectRoot(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/nowhere", nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/nowhere").
		Return("", ports.UnsareportConfig{}, false, nil)

	svc := NewComponentService(
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Remove(context.Background(), "mycomp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsareport.json not found")
}

func TestComponentService_Remove_FileNotExists(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)
	fs.On("FileExists", "/project/components/mycomp.typ").Return(false)

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

	svc := NewComponentService(
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Remove(context.Background(), "mycomp")
	require.NoError(t, err)
}

func TestComponentService_Remove_DeleteError(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)
	fs.On("FileExists", "/project/components/mycomp.typ").Return(true)
	fs.On("Remove", "/project/components/mycomp.typ").Return(fmt.Errorf("permission denied"))

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{
			Components: map[string]ports.ComponentConfigEntry{
				"mycomp": {Version: "1.0.0"},
			},
		}, true, nil)

	svc := NewComponentService(
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Remove(context.Background(), "mycomp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete component file")
}

func TestComponentService_List_Success(t *testing.T) {
	t.Parallel()

	reg := mocks.NewComponentRegistry(t)
	reg.On("ListComponents").Return([]ports.ComponentInfo{
		{Name: "alpha", Description: "First component"},
		{Name: "beta", Description: "Second component"},
	}, nil)

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{
			Components: map[string]ports.ComponentConfigEntry{
				"alpha": {Version: "1.0.0"},
			},
		}, true, nil)

	var stdout bytes.Buffer
	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&stdout),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "NAME")
	assert.Contains(t, stdout.String(), "alpha")
	assert.Contains(t, stdout.String(), "1.0.0")
	assert.Contains(t, stdout.String(), "beta")
	assert.Contains(t, stdout.String(), "no")
}

func TestComponentService_List_RegistryError(t *testing.T) {
	t.Parallel()

	reg := mocks.NewComponentRegistry(t)
	reg.On("ListComponents").Return(nil, fmt.Errorf("network error"))

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(mocks.NewFileSystem(t)),
		WithComponentConfig(mocks.NewConfigStore(t)),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.List(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list components")
}

func TestComponentService_List_NoConfig(t *testing.T) {
	t.Parallel()

	reg := mocks.NewComponentRegistry(t)
	reg.On("ListComponents").Return([]ports.ComponentInfo{
		{Name: "alpha", Description: "First component"},
	}, nil)

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{}, false, nil)

	var stdout bytes.Buffer
	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&stdout),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "alpha")
	assert.Contains(t, stdout.String(), "no")
}

func TestComponentService_Update_AlreadyUpToDate(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	info := &ports.ComponentInfo{Name: "mycomp"}
	cv := &ports.ComponentVersion{Path: "mycomp/1.0.0.typ"}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "mycomp", "latest").Return(v1, info, cv, nil)

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{
			Components: map[string]ports.ComponentConfigEntry{
				"mycomp": {Version: "1.0.0"},
			},
		}, true, nil)

	var stdout bytes.Buffer
	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&stdout),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Update(context.Background(), "mycomp")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "already up to date")
}

func TestComponentService_Update_NotInstalled(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{
			Components: map[string]ports.ComponentConfigEntry{},
		}, true, nil)

	svc := NewComponentService(
		WithComponentRegistry(mocks.NewComponentRegistry(t)),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Update(context.Background(), "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not installed")
}

func TestComponentService_Update_NoProjectRoot(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/nowhere", nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/nowhere").
		Return("", ports.UnsareportConfig{}, false, nil)

	svc := NewComponentService(
		WithComponentRegistry(mocks.NewComponentRegistry(t)),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Update(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsareport.json not found")
}

func TestComponentService_Update_NeedsUpdate(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	v2, _ := semver.NewVersion("2.0.0")
	info := &ports.ComponentInfo{Name: "mycomp"}
	cv := &ports.ComponentVersion{Path: "mycomp/2.0.0.typ"}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "mycomp", "latest").Return(v2, info, cv, nil)
	reg.On("FetchComponentFile", *info, cv).Return([]byte("# mycomp v2"), nil)

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)
	fs.On("FileExists", mock.Anything).Return(false)
	fs.On("EnsureDir", mock.Anything).Return(nil)
	fs.On("WriteFileAtomic", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{
			Components: map[string]ports.ComponentConfigEntry{
				"mycomp": {Version: v1.String()},
			},
		}, true, nil)
	cfg.On("ReadLockfile", "/project").Return(ports.Lockfile{
		Packages: map[string]ports.LockfilePackage{},
	}, nil)
	cfg.On("WriteLockfile", "/project", mock.Anything).Return(nil)
	cfg.On("WriteConfig", "/project", mock.Anything).Return(nil)

	var stdout bytes.Buffer
	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&stdout),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Update(context.Background(), "mycomp")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "updating 1.0.0 -> 2.0.0")
}

func TestComponentService_Update_All_NoComponents(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{
			Components: nil,
		}, true, nil)

	var stdout bytes.Buffer
	svc := NewComponentService(
		WithComponentRegistry(mocks.NewComponentRegistry(t)),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&stdout),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Update(context.Background(), "")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "No components installed")
}

func TestComponentService_AddFromManifest_Success(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	info := &ports.ComponentInfo{Name: "comp-a"}
	cv := &ports.ComponentVersion{Path: "comp-a/1.0.0.typ"}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "comp-a", "^1.0.0").Return(v1, info, cv, nil)
	reg.On("FetchComponentFile", *info, cv).Return([]byte("# comp-a"), nil)

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

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	results, err := svc.AddFromManifest(context.Background(), map[string]string{
		"comp-a": "^1.0.0",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "comp-a", results[0].Name)
	assert.Equal(t, "installed", results[0].Status)
}

func TestComponentService_AddFromManifest_ResolveError(t *testing.T) {
	t.Parallel()

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "bad", "^1.0.0").
		Return(nil, nil, nil, fmt.Errorf("not found"))

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(mocks.NewFileSystem(t)),
		WithComponentConfig(mocks.NewConfigStore(t)),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	_, err := svc.AddFromManifest(context.Background(), map[string]string{
		"bad": "^1.0.0",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve bad")
}

func TestComponentService_Add_WriteLockfileError(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	info := &ports.ComponentInfo{Name: "mycomp"}
	cv := &ports.ComponentVersion{Path: "mycomp/1.0.0.typ"}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "mycomp", "latest").Return(v1, info, cv, nil)
	reg.On("FetchComponentFile", *info, cv).Return([]byte("# mycomp"), nil)

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
	cfg.On("WriteLockfile", "/project", mock.Anything).Return(fmt.Errorf("disk full"))

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Add(context.Background(), "mycomp", "latest", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write lockfile")
}

func TestComponentService_Add_WriteConfigError(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	info := &ports.ComponentInfo{Name: "mycomp"}
	cv := &ports.ComponentVersion{Path: "mycomp/1.0.0.typ"}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "mycomp", "latest").Return(v1, info, cv, nil)
	reg.On("FetchComponentFile", *info, cv).Return([]byte("# mycomp"), nil)

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
	cfg.On("WriteConfig", "/project", mock.Anything).Return(fmt.Errorf("write failed"))

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Add(context.Background(), "mycomp", "latest", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write config")
}

func TestComponentService_Add_EnsureDirError(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	info := &ports.ComponentInfo{Name: "mycomp"}
	cv := &ports.ComponentVersion{Path: "mycomp/1.0.0.typ"}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "mycomp", "latest").Return(v1, info, cv, nil)
	reg.On("FetchComponentFile", *info, cv).Return([]byte("# mycomp"), nil)

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)
	fs.On("FileExists", mock.Anything).Return(false)
	fs.On("EnsureDir", mock.Anything).Return(fmt.Errorf("mkdir failed"))

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{}, true, nil)

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Add(context.Background(), "mycomp", "latest", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ensure components dir")
}

func TestComponentService_Add_WriteFileError(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	info := &ports.ComponentInfo{Name: "mycomp"}
	cv := &ports.ComponentVersion{Path: "mycomp/1.0.0.typ"}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "mycomp", "latest").Return(v1, info, cv, nil)
	reg.On("FetchComponentFile", *info, cv).Return([]byte("# mycomp"), nil)

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)
	fs.On("FileExists", mock.Anything).Return(false)
	fs.On("EnsureDir", mock.Anything).Return(nil)
	fs.On("WriteFileAtomic", mock.Anything, mock.Anything, mock.Anything).
		Return(fmt.Errorf("disk full"))

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{}, true, nil)

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Add(context.Background(), "mycomp", "latest", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write component")
}

func TestComponentService_Add_GetwdError(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	info := &ports.ComponentInfo{Name: "mycomp"}
	cv := &ports.ComponentVersion{Path: "mycomp/1.0.0.typ"}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "mycomp", "latest").Return(v1, info, cv, nil)

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("", fmt.Errorf("getwd failed"))

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(mocks.NewConfigStore(t)),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Add(context.Background(), "mycomp", "latest", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get cwd")
}

func TestComponentService_Remove_GetwdError(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("", fmt.Errorf("getwd failed"))

	svc := NewComponentService(
		WithComponentRegistry(mocks.NewComponentRegistry(t)),
		WithComponentFS(fs),
		WithComponentConfig(mocks.NewConfigStore(t)),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Remove(context.Background(), "mycomp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get cwd")
}

func TestComponentService_List_GetwdError(t *testing.T) {
	t.Parallel()

	reg := mocks.NewComponentRegistry(t)
	reg.On("ListComponents").Return([]ports.ComponentInfo{}, nil)

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("", fmt.Errorf("getwd failed"))

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(mocks.NewConfigStore(t)),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.List(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get cwd")
}

func TestComponentService_Update_GetwdError(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("", fmt.Errorf("getwd failed"))

	svc := NewComponentService(
		WithComponentRegistry(mocks.NewComponentRegistry(t)),
		WithComponentFS(fs),
		WithComponentConfig(mocks.NewConfigStore(t)),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Update(context.Background(), "mycomp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get cwd")
}

func TestComponentService_Update_ResolveLatestError(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{
			Components: map[string]ports.ComponentConfigEntry{
				"mycomp": {Version: "1.0.0"},
			},
		}, true, nil)

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "mycomp", "latest").
		Return(nil, nil, nil, fmt.Errorf("network error"))

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Update(context.Background(), "mycomp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve latest version")
}

func TestComponentService_Add_WriteLockfileReadError(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	info := &ports.ComponentInfo{Name: "mycomp"}
	cv := &ports.ComponentVersion{Path: "mycomp/1.0.0.typ"}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "mycomp", "latest").Return(v1, info, cv, nil)
	reg.On("FetchComponentFile", *info, cv).Return([]byte("# mycomp"), nil)

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
	cfg.On("ReadLockfile", "/project").
		Return(ports.Lockfile{}, fmt.Errorf("read failed"))

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Add(context.Background(), "mycomp", "latest", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read lockfile")
}

func TestComponentService_Remove_LockfileError(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)
	fs.On("FileExists", "/project/components/mycomp.typ").Return(true)
	fs.On("Remove", "/project/components/mycomp.typ").Return(nil)

	cfg := mocks.NewConfigStore(t)
	cfg.On("FindProjectRoot", "/project").
		Return("/project", ports.UnsareportConfig{
			Components: map[string]ports.ComponentConfigEntry{
				"mycomp": {Version: "1.0.0"},
			},
		}, true, nil)
	cfg.On("ReadLockfile", "/project").
		Return(ports.Lockfile{}, fmt.Errorf("read failed"))

	svc := NewComponentService(
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Remove(context.Background(), "mycomp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read lockfile")
}

func TestComponentService_Remove_WriteLockfileError(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)
	fs.On("FileExists", "/project/components/mycomp.typ").Return(true)
	fs.On("Remove", "/project/components/mycomp.typ").Return(nil)

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
	cfg.On("WriteLockfile", "/project", mock.Anything).
		Return(fmt.Errorf("write failed"))

	svc := NewComponentService(
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Remove(context.Background(), "mycomp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write lockfile")
}

func TestComponentService_Remove_WriteConfigError(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem(t)
	fs.On("Getwd").Return("/project", nil)
	fs.On("FileExists", "/project/components/mycomp.typ").Return(true)
	fs.On("Remove", "/project/components/mycomp.typ").Return(nil)

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
	cfg.On("WriteConfig", "/project", mock.Anything).
		Return(fmt.Errorf("write failed"))

	svc := NewComponentService(
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Remove(context.Background(), "mycomp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write config")
}

func TestComponentService_Add_WithCircularDeps(t *testing.T) {
	t.Parallel()

	v1, _ := semver.NewVersion("1.0.0")
	info := &ports.ComponentInfo{Name: "a"}
	cv := &ports.ComponentVersion{Path: "a/1.0.0.typ", Dependencies: []string{"b@^1.0.0"}}
	bInfo := &ports.ComponentInfo{Name: "b"}
	bCv := &ports.ComponentVersion{Path: "b/1.0.0.typ", Dependencies: []string{"a@^1.0.0"}}

	reg := mocks.NewComponentRegistry(t)
	reg.On("ResolveVersion", "a", "latest").Return(v1, info, cv, nil)
	reg.On("ResolveVersion", "b", "^1.0.0").Return(v1, bInfo, bCv, nil)
	reg.On("FetchComponentFile", *info, cv).Return([]byte("# a"), nil)

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

	svc := NewComponentService(
		WithComponentRegistry(reg),
		WithComponentFS(fs),
		WithComponentConfig(cfg),
		WithComponentStdout(&bytes.Buffer{}),
		WithComponentStderr(&bytes.Buffer{}),
	)

	err := svc.Add(context.Background(), "a", "latest", false)
	require.NoError(t, err)
}
