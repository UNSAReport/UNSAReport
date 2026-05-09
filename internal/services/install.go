package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/christianmz565/lab-report/internal/ports"
)

type InstallOptions struct {
	Dest  string
	Multi bool
	Repo  string
	Ref   string
}

type InstallService struct {
	Fetcher ports.TemplateFetcher
	FS      ports.FileSystem
	Config  ports.ConfigStore
}

func NewInstallService(f ports.TemplateFetcher, fs ports.FileSystem, c ports.ConfigStore) *InstallService {
	return &InstallService{
		Fetcher: f,
		FS:      fs,
		Config:  c,
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

	var files map[string][]byte
	if info, err := s.FS.Stat(opt.Repo); err == nil && info.IsDir() {
		files, err = s.Fetcher.LoadLocal(opt.Repo)
		if err != nil {
			return fmt.Errorf("load local templates: %w", err)
		}
	} else {
		files, err = s.Fetcher.Fetch(ctx, opt.Repo, opt.Ref)
		if err != nil {
			return fmt.Errorf("fetch templates: %w", err)
		}
	}

	m, err := LoadManifest(files)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	if err := s.FS.EnsureDir(destDir); err != nil {
		return fmt.Errorf("ensure dest dir: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Installing lab report template to: %s\n", destDir)
	if opt.Multi {
		fmt.Fprintln(os.Stdout, "Mode: Multi-lab (--multi)")
	}
	fmt.Fprintln(os.Stdout, strings.Repeat("-", 50))

	if opt.Multi {
		rootEntries := ExpandDirEntries(files, m.Multi.Root)
		if err := s.applyEntriesInstall(files, destDir, rootEntries); err != nil {
			return err
		}
		if err := s.applyEntryInstall(files, destDir, m.Multi.Readme); err != nil {
			return err
		}

		lab := "l1"
		labEntries := substituteLab(m.Multi.LabFiles, lab)
		labEntriesExpanded := ExpandDirEntries(files, labEntries)
		if err := s.applyEntriesInstall(files, destDir, labEntriesExpanded); err != nil {
			return err
		}
	} else {
		allEntries := ExpandDirEntries(files, append(m.Common, m.Single...))
		if err := s.applyEntriesInstall(files, destDir, allEntries); err != nil {
			return err
		}
	}

	if err := s.Config.WriteConfig(destDir, ports.LabReportConfig{MultiLab: opt.Multi}); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	fmt.Fprintf(os.Stdout, "Created: labreport.json (Mode: %s)\n", map[bool]string{true: "multi", false: "single"}[opt.Multi])

	fmt.Fprintln(os.Stdout, strings.Repeat("-", 50))
	fmt.Fprintln(os.Stdout, "Installation complete!")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Next steps:")
	for _, step := range s.nextSteps(opt.Multi) {
		fmt.Fprintln(os.Stdout, step)
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

func (s *InstallService) nextSteps(multi bool) []string {
	if multi {
		return []string{
			"1. Edit l1/report.typ with your lab information",
			"2. Place your source code in l1/src/",
			"3. Compile the report:",
			"   lab-report prepare l1",
		}
	}
	return []string{
		"1. Edit report.typ with your project information",
		"2. Place your source code in src/",
		"3. Compile the report:",
		"   lab-report prepare",
	}
}
