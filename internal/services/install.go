package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/UNSAReport/UNSAReport/internal/ports"
)

type InstallOptions struct {
	Dest     string
	Session  string
	Repo     string
	Ref      string
	Template string
}

type InstallService struct {
	Fetcher  ports.TemplateFetcher
	FS       ports.FileSystem
	Config   ports.ConfigStore
	Registry ports.TemplateRegistry
}

func NewInstallService(f ports.TemplateFetcher, fs ports.FileSystem, c ports.ConfigStore, r ports.TemplateRegistry) *InstallService {
	return &InstallService{
		Fetcher:  f,
		FS:       fs,
		Config:   c,
		Registry: r,
	}
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
		fmt.Fprintln(os.Stdout, err.Error())
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

	template, err := s.Registry.GetTemplate(opt.Template)
	if err != nil {
		return fmt.Errorf("get template: %w", err)
	}

	var files map[string][]byte
	if template.LocalPath != "" {
		if info, err := s.FS.Stat(template.LocalPath); err == nil && info.IsDir() {
			files, err = s.Fetcher.LoadLocal(template.LocalPath)
			if err != nil {
				return fmt.Errorf("load local templates: %w", err)
			}
		}
	} else if template.Repo != "" {
		files, err = s.Fetcher.Fetch(ctx, template.Repo, template.Ref)
		if err != nil {
			return fmt.Errorf("fetch templates: %w", err)
		}
	} else {
		return fmt.Errorf("template %q has no source", opt.Template)
	}

	m, err := LoadManifest(files)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	if err := s.FS.EnsureDir(destDir); err != nil {
		return fmt.Errorf("ensure dest dir: %w", err)
	}

	if opt.Session != "" {
		fmt.Fprintf(os.Stdout, "Installing session '%s' into multi-lab project: %s\n", opt.Session, destDir)
	} else {
		fmt.Fprintf(os.Stdout, "Installing %s template to: %s\n", template.Name, destDir)
		fmt.Fprintf(os.Stdout, "Mode: %s\n", m.Mode)
	}
	fmt.Fprintln(os.Stdout, strings.Repeat("-", 50))

	cfg.Template = template.Name
	cfg.Mode = m.Mode

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
	}

	if err := s.Config.WriteConfig(destDir, cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if !hasConfig {
		fmt.Fprintf(os.Stdout, "Created: unsareport.json (Mode: %s)\n", cfg.Mode)
	} else {
		fmt.Fprintf(os.Stdout, "Updated: unsareport.json\n")
	}

	fmt.Fprintln(os.Stdout, strings.Repeat("-", 50))
	fmt.Fprintln(os.Stdout, "Installation complete!")
	fmt.Fprintln(os.Stdout)
	if !hasConfig {
		fmt.Fprintln(os.Stdout, "Next steps:")
		for _, step := range s.nextSteps(cfg) {
			fmt.Fprintln(os.Stdout, step)
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
			return nil
		}
		if err := s.FS.WriteFileAtomic(dstPath, data, 0o644); err != nil {
			return fmt.Errorf("write file %s: %w", dstPath, err)
		}
		fmt.Fprintf(os.Stdout, "Created: %s\n", e.Dest)
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
