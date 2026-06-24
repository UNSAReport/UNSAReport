package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/UNSAReport/UNSAReport/internal/adapters/config"
	"github.com/UNSAReport/UNSAReport/internal/ports"
	"github.com/charmbracelet/huh"
)

type UpdateOptions struct {
	Dest     string
	Force    bool
	Session  string
	Local    string
	Rollback bool
}

type UpdateService struct {
	Fetcher          ports.TemplateFetcher
	FS               ports.FileSystem
	Config           ports.ConfigStore
	Registry         ports.TemplateRegistry
	ComponentService *ComponentService
	Stdout           io.Writer
	Stderr           io.Writer
}

func NewUpdateService(f ports.TemplateFetcher, fs ports.FileSystem, c ports.ConfigStore, r ports.TemplateRegistry, cs *ComponentService, stdout, stderr io.Writer) *UpdateService {
	return &UpdateService{
		Fetcher:          f,
		FS:               fs,
		Config:           c,
		Registry:         r,
		ComponentService: cs,
		Stdout:           stdout,
		Stderr:           stderr,
	}
}

type updateDecision string

const (
	decYes  updateDecision = "yes"
	decNo   updateDecision = "no"
	decAll  updateDecision = "all"
	decQuit updateDecision = "quit"
)

var imageExt = regexp.MustCompile(`(?i)\.(png|jpe?g|gif|svg|webp|ico)$`)

func (s *UpdateService) Execute(ctx context.Context, opt UpdateOptions) error {
	destDir := opt.Dest
	if destDir == "" {
		destDir = "."
	}
	destDir, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("abs dest path: %w", err)
	}

	projectRoot, cfg, ok, err := s.Config.FindProjectRoot(destDir)
	if err != nil {
		return fmt.Errorf("find project root: %w", err)
	}

	isMulti := false
	if ok {
		isMulti = cfg.Mode == "multi"
		destDir = projectRoot
	} else {
		return fmt.Errorf("no project found in %s. Run 'unsarep install' first", destDir)
	}

	if err := s.FS.Chdir(destDir); err != nil {
		return fmt.Errorf("chdir to dest: %w", err)
	}

	template, err := s.Registry.GetTemplateVersion(cfg.Template, cfg.TemplateVersion)
	if err != nil {
		return fmt.Errorf("get template: %w", err)
	}

	latestTemplate, err := s.Registry.GetTemplateVersion(cfg.Template, "latest")
	if err == nil && cfg.TemplateVersion != "" {
		current, cErr := semver.NewVersion(cfg.TemplateVersion)
		latest, lErr := semver.NewVersion(latestTemplate.Version)
		if cErr == nil && lErr == nil {
			if latest.GreaterThan(current) {
				if latest.Major() > current.Major() {
					fmt.Fprintf(s.Stdout, "Major version update available: %s -> %s\n", current, latest)
					fmt.Fprintf(s.Stdout, "Run 'unsarep install %s@%s' to upgrade to the new major version.\n", cfg.Template, latest)
					return nil
				}
				fmt.Fprintf(s.Stdout, "Template update available: %s -> %s (current: %s)\n", cfg.Template, latest, current)
				template = latestTemplate
			}
		}
	}

	localSource := opt.Local
	if localSource == "" {
		localSource = cfg.LocalSource
	}

	var remoteFiles map[string][]byte
	if localSource != "" {
		remoteFiles, err = s.Fetcher.LoadLocal(filepath.Join(localSource, template.Path))
		if err != nil {
			return fmt.Errorf("load local templates: %w", err)
		}
	} else {
		remoteFiles, err = s.Fetcher.Fetch(ctx, ports.DefaultTemplateRepo, ports.DefaultRef, template.Path)
		if err != nil {
			return fmt.Errorf("fetch templates: %w", err)
		}
	}

	manifestData, ok := remoteFiles["manifest.json"]
	if !ok {
		return fmt.Errorf("manifest.json not found in template %q", cfg.Template)
	}
	m, err := LoadAndValidateManifest(manifestData)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	fmt.Fprintf(s.Stdout, "Detected %s setup.\n", map[bool]string{true: "multi-lab", false: "single-lab"}[isMulti])

	if isMulti && opt.Session == "" && len(cfg.Sessions) > 1 {
		fmt.Fprintf(s.Stdout, "This will update all registered sessions: %s\n", strings.Join(cfg.Sessions, ", "))
		var confirmed bool
		form := huh.NewForm(huh.NewGroup(
			huh.NewConfirm().
				Title("Continue with updating all sessions?").
				Value(&confirmed),
		))
		if err := form.Run(); err != nil {
			return fmt.Errorf("prompt decision: %w", err)
		}
		if !confirmed {
			fmt.Fprintln(s.Stdout, "Update cancelled.")
			return nil
		}
	}

	fmt.Fprintf(s.Stdout, "Checking for updates in: %s\n\n", destDir)

	entries := s.buildUpdateEntries(m, isMulti, cfg, opt.Session)
	entries = ExpandDirEntries(remoteFiles, entries)

	updatableEntries := make([]Entry, 0)
	for _, e := range entries {
		if e.Updatable {
			updatableEntries = append(updatableEntries, e)
		}
	}

	if len(updatableEntries) > 0 {
		rb := NewRollbackService(s.FS, s.Config, s.Stdout, s.Stderr)
		if err := rb.CreateBackup(destDir, updatableEntries, cfg); err != nil {
			fmt.Fprintf(s.Stderr, "Warning: could not create backup: %v\n", err)
		}
	}

	applied := 0
	autoAcceptAll := opt.Force

	for _, e := range entries {
		if !e.Updatable {
			continue
		}

		dstPath := filepath.Join(destDir, filepath.FromSlash(e.Dest))
		srcPath := e.Src

		switch e.Kind {
		case KindDir:
			if err := s.FS.EnsureDir(dstPath); err != nil {
				return fmt.Errorf("ensure dir: %w", err)
			}
			continue
		case KindFile:
			remote, ok := remoteFiles[srcPath]
			if !ok {
				continue
			}

			local, lerr := s.FS.ReadFile(dstPath)
			isNew := os.IsNotExist(lerr)
			if lerr != nil && !isNew {
				return fmt.Errorf("read local file: %w", lerr)
			}
			if !isNew && s.FS.SameContent(local, remote) {
				continue
			}

			label := e.Dest
			apply := func() error {
				if err := s.FS.WriteFileAtomic(dstPath, remote, 0o644); err != nil {
					return fmt.Errorf("write atomic: %w", err)
				}
				applied++
				return nil
			}

			if autoAcceptAll {
				if err := apply(); err != nil {
					return err
				}
				fmt.Fprintf(s.Stdout, "✓ Updated [FORCED]: %s\n", label)
				continue
			}

			if isNew {
				fmt.Fprintf(s.Stdout, "\n[NEW FILE] %s\n", label)
			} else {
				s.showDiff(label, string(local), string(remote), dstPath, srcPath)
			}

			dec, err := s.promptUpdateDecision(label)
			if err != nil {
				return fmt.Errorf("prompt decision: %w", err)
			}
			switch dec {
			case decAll:
				autoAcceptAll = true
				fallthrough
			case decYes:
				if err := apply(); err != nil {
					return err
				}
				fmt.Fprintf(s.Stdout, "✓ Updated: %s\n", label)
			case decNo:
				fmt.Fprintf(s.Stdout, "Skipped: %s\n", label)
			case decQuit:
				fmt.Fprintln(s.Stdout, "\nStopping update process.")
				return nil
			}
		default:
			return fmt.Errorf("unknown manifest entry kind: %s", e.Kind)
		}
	}

	fmt.Fprintf(s.Stdout, "\nUpdate finished. %d files updated.\n", applied)

	if s.ComponentService != nil {
		if err := s.syncComponents(ctx, m.GetComponents(), cfg); err != nil {
			fmt.Fprintf(s.Stderr, "Warning: component sync failed: %v\n", err)
		}
	}

	cfg.TemplateVersion = template.Version
	if err := s.Config.WriteConfig(destDir, cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	s.recordTemplateLockfile(destDir, cfg, m, remoteFiles)

	return nil
}

func (s *UpdateService) buildUpdateEntries(m *Manifest, isMulti bool, cfg ports.UnsareportConfig, session string) []Entry {
	var out []Entry
	add := func(entries ...[]Entry) {
		for _, list := range entries {
			out = append(out, list...)
		}
	}

	if isMulti {
		multiEntries, err := m.GetMultiEntries()
		if err != nil {
			return nil
		}
		add(multiEntries.Root)

		labs := cfg.Sessions
		if session != "" {
			labs = []string{session}
		}

		for _, lab := range labs {
			out = append(out, substituteLab(multiEntries.LabFiles, lab)...)
		}
	} else {
		singleEntries, err := m.GetSingleEntries()
		if err != nil {
			return nil
		}
		add(singleEntries)
	}

	seen := map[string]Entry{}
	for _, e := range out {
		seen[e.Dest] = e
	}

	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	final := make([]Entry, 0, len(keys))
	for _, k := range keys {
		e := seen[k]
		parts := strings.Split(e.Dest, "/")
		last := parts[len(parts)-1]
		switch last {
		case "report.typ":
			parts[len(parts)-1] = cfg.Prepare.Input.ReportFile
		case "src":
			parts[len(parts)-1] = cfg.Prepare.Input.SrcDir
		case "submission":
			parts[len(parts)-1] = cfg.Prepare.Output.SubmissionDir
		}
		e.Dest = strings.Join(parts, "/")
		final = append(final, e)
	}
	return final
}

func (s *UpdateService) showDiff(fileName, oldText, newText, oldPath, newSrc string) {
	fmt.Fprintf(s.Stdout, "\n--- Diff for %s ---\n", fileName)
	if imageExt.MatchString(fileName) {
		fmt.Fprintln(s.Stdout, "  [Binary image file changed. Cannot display inline diff.]")
		fmt.Fprintf(s.Stdout, "  - Local version at: %s\n", oldPath)
		fmt.Fprintf(s.Stdout, "  + New template at: %s\n", newSrc)
		fmt.Fprintln(s.Stdout, strings.Repeat("-", 30)+"\n")
		return
	}

	fmt.Fprint(s.Stdout, UnifiedLineDiff(oldText, newText))
	fmt.Fprintln(s.Stdout, strings.Repeat("-", 30)+"\n")
}

func (s *UpdateService) promptUpdateDecision(label string) (updateDecision, error) {
	var choice updateDecision
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[updateDecision]().
				Title(fmt.Sprintf("Apply update to %s?", label)).
				Options(
					huh.NewOption("Yes", decYes),
					huh.NewOption("No", decNo),
					huh.NewOption("All remaining", decAll),
					huh.NewOption("Quit", decQuit),
				).
				Value(&choice),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	return choice, nil
}

func (s *UpdateService) recordTemplateLockfile(destDir string, cfg ports.UnsareportConfig, m *Manifest, remoteFiles map[string][]byte) {
	templateFiles := make(map[string]ports.LockfileTemplateFile)

	entries := s.buildUpdateEntries(m, cfg.Mode == "multi", cfg, "")
	expanded := ExpandDirEntries(remoteFiles, entries)

	for _, entry := range expanded {
		if entry.Kind != KindFile {
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
		templateFiles[entry.Dest] = ports.LockfileTemplateFile{
			Integrity: config.ComputeIntegrity(localData),
		}
	}

	lf, err := s.Config.ReadLockfile(destDir)
	if err != nil {
		lf = ports.Lockfile{}
	}

	lf.Template = &ports.LockfileTemplate{
		Name:    cfg.Template,
		Version: cfg.TemplateVersion,
		Files:   templateFiles,
	}

	if err := s.Config.WriteLockfile(destDir, lf); err != nil {
		fmt.Fprintf(s.Stderr, "Warning: failed to write lockfile: %v\n", err)
	}
}

func (s *UpdateService) syncComponents(ctx context.Context, manifestComponents map[string]string, cfg ports.UnsareportConfig) error {
	if len(manifestComponents) == 0 {
		return nil
	}

	fmt.Fprintf(s.Stdout, "\nSyncing components...\n")
	fmt.Fprintln(s.Stdout, strings.Repeat("-", 50))

	installed := make(map[string]string)
	if cfg.Components != nil {
		for name, entry := range cfg.Components {
			installed[name] = entry.Version
		}
	}

	for name, rangeSpec := range manifestComponents {
		installedVersion, isInstalled := installed[name]

		if isInstalled {
			constraint, err := semver.NewConstraint(rangeSpec)
			if err != nil {
				continue
			}
			v, err := semver.NewVersion(installedVersion)
			if err != nil {
				continue
			}
			if ok, _ := constraint.Validate(v); ok {
				continue
			}

			fmt.Fprintf(s.Stdout, "  Updating %s: %s -> %s\n", name, installedVersion, rangeSpec)
			if err := s.ComponentService.Add(ctx, name, rangeSpec, false); err != nil {
				fmt.Fprintf(s.Stderr, "  Warning: failed to update %s: %v\n", name, err)
			}
		} else {
			fmt.Fprintf(s.Stdout, "  Installing %s (%s)\n", name, rangeSpec)
			if err := s.ComponentService.Add(ctx, name, rangeSpec, false); err != nil {
				fmt.Fprintf(s.Stderr, "  Warning: failed to install %s: %v\n", name, err)
			}
		}
	}

	for name := range installed {
		if _, required := manifestComponents[name]; !required {
			fmt.Fprintf(s.Stdout, "  Note: Component %s is installed but not required by the template\n", name)
		}
	}

	fmt.Fprintln(s.Stdout, strings.Repeat("-", 50))
	return nil
}
