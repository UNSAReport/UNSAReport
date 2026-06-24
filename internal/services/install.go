package services

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	"github.com/UNSAReport/UNSAReport/internal/adapters/config"
	"github.com/UNSAReport/UNSAReport/internal/ports"
)

type InstallOptions struct {
	Dest     string
	Session  string
	Template string
	Local    string
}

type InstallService struct {
	Fetcher          ports.TemplateFetcher
	FS               ports.FileSystem
	Config           ports.ConfigStore
	Registry         ports.TemplateRegistry
	ComponentService *ComponentService
	Stdout           io.Writer
	Stderr           io.Writer
}

type InstallOption func(*InstallService)

func WithInstallFetcher(f ports.TemplateFetcher) InstallOption {
	return func(s *InstallService) { s.Fetcher = f }
}

func WithInstallFS(fs ports.FileSystem) InstallOption {
	return func(s *InstallService) { s.FS = fs }
}

func WithInstallConfig(c ports.ConfigStore) InstallOption {
	return func(s *InstallService) { s.Config = c }
}

func WithInstallRegistry(r ports.TemplateRegistry) InstallOption {
	return func(s *InstallService) { s.Registry = r }
}

func WithInstallComponentService(cs *ComponentService) InstallOption {
	return func(s *InstallService) { s.ComponentService = cs }
}

func WithInstallStdout(w io.Writer) InstallOption {
	return func(s *InstallService) { s.Stdout = w }
}

func WithInstallStderr(w io.Writer) InstallOption {
	return func(s *InstallService) { s.Stderr = w }
}

func NewInstallService(opts ...InstallOption) *InstallService {
	s := &InstallService{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *InstallService) Execute(ctx context.Context, opt InstallOptions) error {
	destDir := opt.Dest
	if destDir == "" {
		destDir = "."
	}
	destDir, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("abs dest path: %w", err)
	}

	projectRoot, cfg, hasConfig, err := s.Config.FindProjectRoot(destDir)
	if err != nil {
		return fmt.Errorf("find project root: %w", err)
	}
	if hasConfig {
		destDir = projectRoot
	} else {
		cfg = ports.UnsareportConfig{
			Mode:     "",
			Sessions: []string{},
			Prepare: ports.PrepareConfig{
				Input: ports.PrepareInputConfig{
					SrcDir:     "src",
					ReportFile: "report.typ",
				},
				Output: ports.PrepareOutputConfig{
					SubmissionDir: "submission",
					FileTemplate:  "{output_type}_{lab_number}",
					ReportWord:    "Informe",
					CodeWord:      "Código Fuente",
				},
			},
			Capture: ports.CaptureConfig{
				Columns:     120,
				FreezeFlags: []string{},
				Prompt:      "❯ ",
				Colors: map[string]string{
					"prompt":  "32",
					"command": "36",
					"args":    "33",
					"reset":   "0",
				},
			},
		}
	}

	if opt.Session != "" && cfg.Mode != "multi" {
		return fmt.Errorf("--session flag can only be used with multi-mode templates")
	}

	name, rangeSpec := parseTemplateArg(opt.Template)

	template, err := s.Registry.GetTemplateVersion(name, rangeSpec)
	if err != nil {
		return fmt.Errorf("get template: %w", err)
	}

	var files map[string][]byte
	if opt.Local != "" {
		localDir := filepath.Join(opt.Local, template.Path)
		files, err = s.Fetcher.LoadLocal(localDir)
		if err != nil {
			return fmt.Errorf("load local templates: %w", err)
		}
		cfg.LocalSource = opt.Local
	} else {
		files, err = s.Fetcher.Fetch(ctx, ports.DefaultTemplateRepo, ports.DefaultRef, template.Path)
		if err != nil {
			return fmt.Errorf("fetch templates: %w", err)
		}
	}

	manifestData, ok := files["manifest.json"]
	if !ok {
		return fmt.Errorf("manifest.json not found in template %q", template.Name)
	}
	m, err := LoadAndValidateManifest(manifestData)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	if err := s.FS.EnsureDir(destDir); err != nil {
		return fmt.Errorf("ensure dest dir: %w", err)
	}

	if opt.Session != "" {
		if _, err := fmt.Fprintf(s.Stdout, "Installing session '%s' into multi-lab project: %s\n", opt.Session, destDir); err != nil {
			return fmt.Errorf("write message: %w", err)
		}
	} else {
		if _, err := fmt.Fprintf(s.Stdout, "Installing %s template to: %s\n", template.Name, destDir); err != nil {
			return fmt.Errorf("write message: %w", err)
		}
		if _, err := fmt.Fprintf(s.Stdout, "Mode: %s\n", m.Mode); err != nil {
			return fmt.Errorf("write message: %w", err)
		}
	}
	if _, err := fmt.Fprintln(s.Stdout, strings.Repeat("-", 50)); err != nil {
		return fmt.Errorf("write separator: %w", err)
	}

	cfg.Template = template.Name
	cfg.TemplateVersion = template.Version
	cfg.Mode = m.Mode

	var installedEntries []Entry
	if m.Mode == "multi" {
		if !hasConfig {
			multiEntries, err := m.GetMultiEntries()
			if err != nil {
				return fmt.Errorf("get multi entries: %w", err)
			}
			rootEntries := ExpandDirEntries(files, multiEntries.Root)
			if err := s.applyEntriesInstall(files, destDir, rootEntries); err != nil {
				return err
			}
			installedEntries = append(installedEntries, rootEntries...)
		}

		multiEntries, err := m.GetMultiEntries()
		if err != nil {
			return fmt.Errorf("get multi entries: %w", err)
		}

		lab := "l1"
		if opt.Session != "" {
			lab = opt.Session
		}

		labEntries := substituteLab(multiEntries.LabFiles, lab)
		labEntriesExpanded := ExpandDirEntries(files, labEntries)
		if err := s.applyEntriesInstall(files, destDir, labEntriesExpanded); err != nil {
			return err
		}
		installedEntries = append(installedEntries, labEntriesExpanded...)

		sessionFound := slices.Contains(cfg.Sessions, lab)
		if !sessionFound {
			cfg.Sessions = append(cfg.Sessions, lab)
		}

	} else {
		singleEntries, err := m.GetSingleEntries()
		if err != nil {
			return fmt.Errorf("get single entries: %w", err)
		}
		allEntries := ExpandDirEntries(files, singleEntries)
		if err := s.applyEntriesInstall(files, destDir, allEntries); err != nil {
			return err
		}
		installedEntries = append(installedEntries, allEntries...)
	}

	if err := s.Config.WriteConfig(destDir, cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if !hasConfig {
		if _, err := fmt.Fprintf(s.Stdout, "Created: unsareport.json (Mode: %s)\n", cfg.Mode); err != nil {
			return fmt.Errorf("write message: %w", err)
		}
	} else {
		if _, err := fmt.Fprintln(s.Stdout, "Updated: unsareport.json"); err != nil {
			return fmt.Errorf("write message: %w", err)
		}
	}

	components := m.GetComponents()
	if len(components) > 0 && s.ComponentService != nil {
		if err := s.FS.Chdir(destDir); err != nil {
			return fmt.Errorf("chdir to dest: %w", err)
		}

		if _, err := fmt.Fprintf(s.Stdout, "\n%s\n", "Downloading components..."); err != nil {
			return fmt.Errorf("write message: %w", err)
		}
		if _, err := fmt.Fprintln(s.Stdout, strings.Repeat("-", 50)); err != nil {
			return fmt.Errorf("write separator: %w", err)
		}

		for name, rangeSpec := range components {
			if _, err := fmt.Fprintf(s.Stdout, "  Component: %s (range: %s)\n", name, rangeSpec); err != nil {
				return fmt.Errorf("write component: %w", err)
			}
		}

		results, err := s.ComponentService.AddFromManifest(ctx, components)
		if err != nil {
			return fmt.Errorf("download components: %w", err)
		}

		for _, r := range results {
			if _, err := fmt.Fprintf(s.Stdout, "    -> Resolved: %s (version: %s)\n", r.Name, r.ResolvedVersion); err != nil {
				return fmt.Errorf("write resolved: %w", err)
			}
			if _, err := fmt.Fprintf(s.Stdout, "  Created: components/%s.typ\n", r.Name); err != nil {
				return fmt.Errorf("write created: %w", err)
			}
		}

		if _, err := fmt.Fprintln(s.Stdout, strings.Repeat("-", 50)); err != nil {
			return fmt.Errorf("write separator: %w", err)
		}
		if _, err := fmt.Fprintf(s.Stdout, "Components installed: %d\n", len(results)); err != nil {
			return fmt.Errorf("write count: %w", err)
		}
	}

	s.recordTemplateLockfile(destDir, cfg, installedEntries, files)

	if _, err := fmt.Fprintln(s.Stdout, strings.Repeat("-", 50)); err != nil {
		return fmt.Errorf("write separator: %w", err)
	}
	if _, err := fmt.Fprintln(s.Stdout, "Installation complete!"); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if _, err := fmt.Fprintln(s.Stdout); err != nil {
		return fmt.Errorf("write newline: %w", err)
	}
	if !hasConfig {
		if _, err := fmt.Fprintln(s.Stdout, "Next steps:"); err != nil {
			return fmt.Errorf("write message: %w", err)
		}
		for _, step := range s.nextSteps(cfg) {
			if _, err := fmt.Fprintln(s.Stdout, step); err != nil {
				return fmt.Errorf("write step: %w", err)
			}
		}
	}

	return nil
}

func (s *InstallService) applyEntriesInstall(files map[string][]byte, destDir string, entries []Entry) error {
	for _, e := range entries {
		if err := s.applyEntryInstall(files, destDir, e); err != nil {
			return err
		}
	}
	return nil
}

func (s *InstallService) applyEntryInstall(files map[string][]byte, destDir string, e Entry) error {
	dstPath := filepath.Join(destDir, filepath.FromSlash(e.Dest))

	switch e.Kind {
	case KindDir:
		if err := s.FS.EnsureDir(dstPath); err != nil {
			return fmt.Errorf("ensure dir %s: %w", dstPath, err)
		}
		return nil
	case KindFile:
		data, ok := files[e.Src]
		if !ok {
			return nil
		}
		if s.FS.FileExists(dstPath) {
			if _, err := fmt.Fprintf(s.Stdout, "Skipped (exists): %s\n", e.Dest); err != nil {
				return fmt.Errorf("write message: %w", err)
			}
			return nil
		}
		if err := s.FS.WriteFileAtomic(dstPath, data, 0o644); err != nil {
			return fmt.Errorf("write file %s: %w", dstPath, err)
		}
		if _, err := fmt.Fprintf(s.Stdout, "Created: %s\n", e.Dest); err != nil {
			return fmt.Errorf("write message: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unknown manifest entry kind: %s", e.Kind)
	}
}

func substituteLab(entries []Entry, lab string) []Entry {
	out := make([]Entry, len(entries))
	for i, e := range entries {
		e.Dest = strings.ReplaceAll(e.Dest, "{lab}", lab)
		out[i] = e
	}
	return out
}

func (s *InstallService) nextSteps(cfg ports.UnsareportConfig) []string {
	if cfg.Mode == "multi" {
		return []string{
			fmt.Sprintf("1. Edit l1/%s with your lab information", cfg.Prepare.Input.ReportFile),
			fmt.Sprintf("2. Place your source code in l1/%s/", cfg.Prepare.Input.SrcDir),
			"3. Compile the report:",
			"   unsarep prepare l1",
		}
	}
	return []string{
		fmt.Sprintf("1. Edit %s with your project information", cfg.Prepare.Input.ReportFile),
		fmt.Sprintf("2. Place your source code in %s/", cfg.Prepare.Input.SrcDir),
		"3. Compile the report:",
		"   unsarep prepare",
	}
}

func parseTemplateArg(arg string) (name, rangeSpec string) {
	if i := strings.Index(arg, "@"); i != -1 {
		return arg[:i], arg[i+1:]
	}
	return arg, "latest"
}

func (s *InstallService) recordTemplateLockfile(destDir string, cfg ports.UnsareportConfig, installedEntries []Entry, remoteFiles map[string][]byte) {
	if len(installedEntries) == 0 {
		return
	}

	lf, err := s.Config.ReadLockfile(destDir)
	if err != nil {
		lf = ports.Lockfile{}
	}

	var templateFiles map[string]ports.LockfileTemplateFile
	if lf.Template != nil && lf.Template.Name == cfg.Template && lf.Template.Version == cfg.TemplateVersion {
		templateFiles = lf.Template.Files
	}
	if templateFiles == nil {
		templateFiles = make(map[string]ports.LockfileTemplateFile)
	}

	for _, entry := range installedEntries {
		if entry.Kind != KindFile {
			continue
		}
		data, ok := remoteFiles[entry.Src]
		if !ok {
			continue
		}
		dstPath := filepath.Join(destDir, filepath.FromSlash(entry.Dest))
		if !s.FS.FileExists(dstPath) {
			continue
		}
		localData, err := s.FS.ReadFile(dstPath)
		if err != nil {
			continue
		}
		if !s.FS.SameContent(localData, data) {
			localData = data
		}
		templateFiles[entry.Dest] = ports.LockfileTemplateFile{
			Integrity: config.ComputeIntegrity(localData),
		}
	}

	lf.Template = &ports.LockfileTemplate{
		Name:    cfg.Template,
		Version: cfg.TemplateVersion,
		Files:   templateFiles,
	}

	if err := s.Config.WriteLockfile(destDir, lf); err != nil {
		slog.Warn("failed to write lockfile", "error", err)
	}
}
