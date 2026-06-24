package mocks

import (
	"context"
	"os"

	"github.com/Masterminds/semver/v3"
	"github.com/UNSAReport/UNSAReport/internal/ports"
)

type MockFileSystem struct {
	EnsureDirFn       func(path string) error
	FileExistsFn      func(path string) bool
	ReadFileFn        func(path string) ([]byte, error)
	WriteFileAtomicFn func(path string, data []byte, perm os.FileMode) error
	CopyFileFn        func(src string, dst string, perm os.FileMode) error
	SameContentFn     func(a, b []byte) bool
	ReadDirFn         func(dirname string) ([]os.DirEntry, error)
	ChdirFn           func(dir string) error
	GetwdFn           func() (string, error)
	RemoveFn          func(path string) error
	StatFn            func(name string) (os.FileInfo, error)
}

func (m *MockFileSystem) EnsureDir(path string) error {
	if m.EnsureDirFn != nil {
		return m.EnsureDirFn(path)
	}
	return nil
}

func (m *MockFileSystem) FileExists(path string) bool {
	if m.FileExistsFn != nil {
		return m.FileExistsFn(path)
	}
	return false
}

func (m *MockFileSystem) ReadFile(path string) ([]byte, error) {
	if m.ReadFileFn != nil {
		return m.ReadFileFn(path)
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	if m.WriteFileAtomicFn != nil {
		return m.WriteFileAtomicFn(path, data, perm)
	}
	return nil
}

func (m *MockFileSystem) CopyFile(src string, dst string, perm os.FileMode) error {
	if m.CopyFileFn != nil {
		return m.CopyFileFn(src, dst, perm)
	}
	return nil
}

func (m *MockFileSystem) SameContent(a, b []byte) bool {
	if m.SameContentFn != nil {
		return m.SameContentFn(a, b)
	}
	return false
}

func (m *MockFileSystem) ReadDir(dirname string) ([]os.DirEntry, error) {
	if m.ReadDirFn != nil {
		return m.ReadDirFn(dirname)
	}
	return nil, nil
}

func (m *MockFileSystem) Chdir(dir string) error {
	if m.ChdirFn != nil {
		return m.ChdirFn(dir)
	}
	return nil
}

func (m *MockFileSystem) Getwd() (string, error) {
	if m.GetwdFn != nil {
		return m.GetwdFn()
	}
	return "/tmp/test", nil
}

func (m *MockFileSystem) Remove(path string) error {
	if m.RemoveFn != nil {
		return m.RemoveFn(path)
	}
	return nil
}

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if m.StatFn != nil {
		return m.StatFn(name)
	}
	return nil, os.ErrNotExist
}

type MockConfigStore struct {
	FindProjectRootFn func(startDir string) (string, ports.UnsareportConfig, bool, error)
	ReadConfigFn      func(destDir string) (ports.UnsareportConfig, bool, error)
	WriteConfigFn     func(destDir string, cfg ports.UnsareportConfig) error
	ReadLockfileFn    func(destDir string) (ports.Lockfile, error)
	WriteLockfileFn   func(destDir string, lf ports.Lockfile) error
}

func (m *MockConfigStore) FindProjectRoot(startDir string) (string, ports.UnsareportConfig, bool, error) {
	if m.FindProjectRootFn != nil {
		return m.FindProjectRootFn(startDir)
	}
	return startDir, ports.UnsareportConfig{}, false, nil
}

func (m *MockConfigStore) ReadConfig(destDir string) (ports.UnsareportConfig, bool, error) {
	if m.ReadConfigFn != nil {
		return m.ReadConfigFn(destDir)
	}
	return ports.UnsareportConfig{}, false, nil
}

func (m *MockConfigStore) WriteConfig(destDir string, cfg ports.UnsareportConfig) error {
	if m.WriteConfigFn != nil {
		return m.WriteConfigFn(destDir, cfg)
	}
	return nil
}

func (m *MockConfigStore) ReadLockfile(destDir string) (ports.Lockfile, error) {
	if m.ReadLockfileFn != nil {
		return m.ReadLockfileFn(destDir)
	}
	return ports.Lockfile{Packages: make(map[string]ports.LockfilePackage)}, nil
}

func (m *MockConfigStore) WriteLockfile(destDir string, lf ports.Lockfile) error {
	if m.WriteLockfileFn != nil {
		return m.WriteLockfileFn(destDir, lf)
	}
	return nil
}

type MockTemplateFetcher struct {
	FetchFn     func(ctx context.Context, repo, ref, templatePath string) (map[string][]byte, error)
	FetchRawFn  func(ctx context.Context, repo, ref, path string) ([]byte, error)
	LoadLocalFn func(dir string) (map[string][]byte, error)
}

func (m *MockTemplateFetcher) Fetch(ctx context.Context, repo, ref, templatePath string) (map[string][]byte, error) {
	if m.FetchFn != nil {
		return m.FetchFn(ctx, repo, ref, templatePath)
	}
	return nil, nil
}

func (m *MockTemplateFetcher) FetchRaw(ctx context.Context, repo, ref, path string) ([]byte, error) {
	if m.FetchRawFn != nil {
		return m.FetchRawFn(ctx, repo, ref, path)
	}
	return nil, nil
}

func (m *MockTemplateFetcher) LoadLocal(dir string) (map[string][]byte, error) {
	if m.LoadLocalFn != nil {
		return m.LoadLocalFn(dir)
	}
	return nil, nil
}

type MockTemplateRegistry struct {
	ListTemplatesFn      func() ([]ports.TemplateInfo, error)
	GetTemplateFn        func(name string) (ports.TemplateInfo, error)
	GetTemplateVersionFn func(name string, rangeSpec string) (ports.TemplateInfo, error)
}

func (m *MockTemplateRegistry) ListTemplates() ([]ports.TemplateInfo, error) {
	if m.ListTemplatesFn != nil {
		return m.ListTemplatesFn()
	}
	return nil, nil
}

func (m *MockTemplateRegistry) GetTemplate(name string) (ports.TemplateInfo, error) {
	if m.GetTemplateFn != nil {
		return m.GetTemplateFn(name)
	}
	return ports.TemplateInfo{}, nil
}

func (m *MockTemplateRegistry) GetTemplateVersion(name string, rangeSpec string) (ports.TemplateInfo, error) {
	if m.GetTemplateVersionFn != nil {
		return m.GetTemplateVersionFn(name, rangeSpec)
	}
	return ports.TemplateInfo{}, nil
}

type MockCompiler struct {
	QueryVarsFn func(ctx context.Context, reportPath string) (map[string]string, error)
	CompileFn   func(ctx context.Context, reportPath, reportPDF string, inputs map[string]string) error
}

func (m *MockCompiler) QueryVars(ctx context.Context, reportPath string) (map[string]string, error) {
	if m.QueryVarsFn != nil {
		return m.QueryVarsFn(ctx, reportPath)
	}
	return nil, nil
}

func (m *MockCompiler) Compile(ctx context.Context, reportPath, reportPDF string, inputs map[string]string) error {
	if m.CompileFn != nil {
		return m.CompileFn(ctx, reportPath, reportPDF, inputs)
	}
	return nil
}

type MockArchiver struct {
	ArchiveDirFn   func(zipPath, srcDir string) error
	ArchiveFilesFn func(zipPath, baseDir string, files []string) error
}

func (m *MockArchiver) ArchiveDir(zipPath, srcDir string) error {
	if m.ArchiveDirFn != nil {
		return m.ArchiveDirFn(zipPath, srcDir)
	}
	return nil
}

func (m *MockArchiver) ArchiveFiles(zipPath, baseDir string, files []string) error {
	if m.ArchiveFilesFn != nil {
		return m.ArchiveFilesFn(zipPath, baseDir, files)
	}
	return nil
}

type MockRenderer struct {
	RenderFn func(ctx context.Context, resultPath string, commands []ports.CaptureCommand, flags []string, cfg ports.CaptureConfig) (string, error)
}

func (m *MockRenderer) Render(ctx context.Context, resultPath string, commands []ports.CaptureCommand, flags []string, cfg ports.CaptureConfig) (string, error) {
	if m.RenderFn != nil {
		return m.RenderFn(ctx, resultPath, commands, flags, cfg)
	}
	return "", nil
}

type MockComponentRegistry struct {
	ListComponentsFn     func() ([]ports.ComponentInfo, error)
	GetComponentFn       func(name string) (ports.ComponentInfo, error)
	ResolveVersionFn     func(name string, rangeSpec string) (*semver.Version, *ports.ComponentInfo, *ports.ComponentVersion, error)
	FetchComponentFileFn func(info ports.ComponentInfo, cv *ports.ComponentVersion) ([]byte, error)
}

func (m *MockComponentRegistry) ListComponents() ([]ports.ComponentInfo, error) {
	if m.ListComponentsFn != nil {
		return m.ListComponentsFn()
	}
	return nil, nil
}

func (m *MockComponentRegistry) GetComponent(name string) (ports.ComponentInfo, error) {
	if m.GetComponentFn != nil {
		return m.GetComponentFn(name)
	}
	return ports.ComponentInfo{}, nil
}

func (m *MockComponentRegistry) ResolveVersion(name string, rangeSpec string) (*semver.Version, *ports.ComponentInfo, *ports.ComponentVersion, error) {
	if m.ResolveVersionFn != nil {
		return m.ResolveVersionFn(name, rangeSpec)
	}
	return nil, nil, nil, nil
}

func (m *MockComponentRegistry) FetchComponentFile(info ports.ComponentInfo, cv *ports.ComponentVersion) ([]byte, error) {
	if m.FetchComponentFileFn != nil {
		return m.FetchComponentFileFn(info, cv)
	}
	return nil, nil
}
