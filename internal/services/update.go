package services

import (
	"context"
	"fmt"
	"io"
	"log/slog"
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

// UpdateOptions holds the parameters for a single template update execution.
type UpdateOptions struct {
	Dest     string
	Force    bool
	Session  string
	Local    string
	Rollback bool
}

// UpdateService manages checking for and applying template updates with interactive conflict resolution.
type UpdateService struct {
	Fetcher          ports.TemplateFetcher
	FS               ports.FileSystem
	Config           ports.ConfigStore
	Registry         ports.TemplateRegistry
	ComponentService *ComponentService
	Stdout           io.Writer
	Stderr           io.Writer
}

// UpdateOption configures an UpdateService via functional options.
type UpdateOption func(*UpdateService)

// WithUpdateFetcher sets the template fetcher used to download updated template files.
func WithUpdateFetcher(f ports.TemplateFetcher) UpdateOption {
	return func(s *UpdateService) { s.Fetcher = f }
}

// WithUpdateFS sets the filesystem used for reading and writing template files.
func WithUpdateFS(fs ports.FileSystem) UpdateOption {
	return func(s *UpdateService) { s.FS = fs }
}

// WithUpdateConfig sets the configuration store for reading and writing project config.
func WithUpdateConfig(c ports.ConfigStore) UpdateOption {
	return func(s *UpdateService) { s.Config = c }
}

// WithUpdateRegistry sets the template registry for resolving template versions.
func WithUpdateRegistry(r ports.TemplateRegistry) UpdateOption {
	return func(s *UpdateService) { s.Registry = r }
}

// WithUpdateComponentService sets the component service for syncing template dependencies.
func WithUpdateComponentService(cs *ComponentService) UpdateOption {
	return func(s *UpdateService) { s.ComponentService = cs }
}

// WithUpdateStdout sets the writer for standard output messages.
func WithUpdateStdout(w io.Writer) UpdateOption {
	return func(s *UpdateService) { s.Stdout = w }
}

// WithUpdateStderr sets the writer for standard error messages.
func WithUpdateStderr(w io.Writer) UpdateOption {
	return func(s *UpdateService) { s.Stderr = w }
}

// NewUpdateService creates an UpdateService with the given functional options applied.
func NewUpdateService(opts ...UpdateOption) *UpdateService {
	s := &UpdateService{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type updateDecision string

const (
	decYes  updateDecision = "yes"
	decNo   updateDecision = "no"
	decAll  updateDecision = "all"
	decQuit updateDecision = "quit"
)

var imageExt = regexp.MustCompile(`(?i)\.(png|jpe?g|gif|svg|webp|ico)$`)

// Execute runs the template update: compares remote files with local copies and prompts for each change.
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
	if !ok {
		return fmt.Errorf("no project found in %s. Run 'unsarep install' first", destDir)
	}

	isMulti := cfg.Mode == "multi"
	destDir = projectRoot

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
					if _, err := fmt.Fprintf(s.Stdout, "Major version update available: %s -> %s\n", current, latest); err != nil {
						return fmt.Errorf("write message: %w", err)
					}
					if _, err := fmt.Fprintf(s.Stdout, "Run 'unsarep install %s@%s' to upgrade to the new major version.\n", cfg.Template, latest); err != nil {
						return fmt.Errorf("write message: %w", err)
					}
					return nil
				}
				if _, err := fmt.Fprintf(s.Stdout, "Template update available: %s -> %s (current: %s)\n", cfg.Template, latest, current); err != nil {
					return fmt.Errorf("write message: %w", err)
				}
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

	setupLabel := map[bool]string{true: "multi-lab", false: "single-lab"}[isMulti]
	if _, err := fmt.Fprintf(s.Stdout, "Detected %s setup.\n", setupLabel); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	needsSessionConfirmation := isMulti && opt.Session == "" && len(cfg.Sessions) > 1
	if needsSessionConfirmation {
		if _, err := fmt.Fprintf(s.Stdout, "This will update all registered sessions: %s\n", strings.Join(cfg.Sessions, ", ")); err != nil {
			return fmt.Errorf("write message: %w", err)
		}
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
			if _, err := fmt.Fprintln(s.Stdout, "Update cancelled."); err != nil {
				return fmt.Errorf("write message: %w", err)
			}
			return nil
		}
	}

	if _, err := fmt.Fprintf(s.Stdout, "Checking for updates in: %s\n\n", destDir); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

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
			slog.Warn("could not create backup", "error", err)
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
				if _, err := fmt.Fprintf(s.Stdout, "✓ Updated [FORCED]: %s\n", label); err != nil {
					return fmt.Errorf("write message: %w", err)
				}
				continue
			}

			if isNew {
				if _, err := fmt.Fprintf(s.Stdout, "\n[NEW FILE] %s\n", label); err != nil {
					return fmt.Errorf("write message: %w", err)
				}
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
				if _, err := fmt.Fprintf(s.Stdout, "✓ Updated: %s\n", label); err != nil {
					return fmt.Errorf("write message: %w", err)
				}
			case decNo:
				if _, err := fmt.Fprintf(s.Stdout, "Skipped: %s\n", label); err != nil {
					return fmt.Errorf("write message: %w", err)
				}
			case decQuit:
				if _, err := fmt.Fprintln(s.Stdout, "\nStopping update process."); err != nil {
					return fmt.Errorf("write message: %w", err)
				}
				return nil
			}
		default:
			return fmt.Errorf("unknown manifest entry kind: %s", e.Kind)
		}
	}

	if _, err := fmt.Fprintf(s.Stdout, "\nUpdate finished. %d files updated.\n", applied); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	if s.ComponentService != nil {
		if err := s.syncComponents(ctx, m.GetComponents(), cfg); err != nil {
			slog.Warn("component sync failed", "error", err)
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
	if _, err := fmt.Fprintf(s.Stdout, "\n--- Diff for %s ---\n", fileName); err != nil {
		slog.Warn("write diff header", "error", err)
		return
	}
	if imageExt.MatchString(fileName) {
		if _, err := fmt.Fprintln(s.Stdout, "  [Binary image file changed. Cannot display inline diff.]"); err != nil {
			slog.Warn("write message", "error", err)
			return
		}
		if _, err := fmt.Fprintf(s.Stdout, "  - Local version at: %s\n", oldPath); err != nil {
			slog.Warn("write message", "error", err)
			return
		}
		if _, err := fmt.Fprintf(s.Stdout, "  + New template at: %s\n", newSrc); err != nil {
			slog.Warn("write message", "error", err)
			return
		}
		if _, err := fmt.Fprintln(s.Stdout, strings.Repeat("-", 30)+"\n"); err != nil {
			slog.Warn("write separator", "error", err)
			return
		}
		return
	}

	if _, err := fmt.Fprint(s.Stdout, UnifiedLineDiff(oldText, newText)); err != nil {
		slog.Warn("write diff", "error", err)
		return
	}
	if _, err := fmt.Fprintln(s.Stdout, strings.Repeat("-", 30)+"\n"); err != nil {
		slog.Warn("write separator", "error", err)
		return
	}
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
		slog.Warn("failed to write lockfile", "error", err)
	}
}

func (s *UpdateService) syncComponents(ctx context.Context, manifestComponents map[string]string, cfg ports.UnsareportConfig) error {
	if len(manifestComponents) == 0 {
		return nil
	}

	if _, err := fmt.Fprintf(s.Stdout, "\nSyncing components...\n"); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if _, err := fmt.Fprintln(s.Stdout, strings.Repeat("-", 50)); err != nil {
		return fmt.Errorf("write separator: %w", err)
	}

	installed := make(map[string]string)
	if cfg.Components != nil {
		for name, entry := range cfg.Components {
			installed[name] = entry.Version
		}
	}

	for name, rangeSpec := range manifestComponents {
		installedVersion, isInstalled := installed[name]

		if !isInstalled {
			if _, err := fmt.Fprintf(s.Stdout, "  Installing %s (%s)\n", name, rangeSpec); err != nil {
				return fmt.Errorf("write message: %w", err)
			}
			if err := s.ComponentService.Add(ctx, name, rangeSpec, false); err != nil {
				slog.Warn("failed to install component", "component", name, "error", err)
			}
			continue
		}

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

		if _, err := fmt.Fprintf(s.Stdout, "  Updating %s: %s -> %s\n", name, installedVersion, rangeSpec); err != nil {
			return fmt.Errorf("write message: %w", err)
		}
		if err := s.ComponentService.Add(ctx, name, rangeSpec, false); err != nil {
			slog.Warn("failed to update component", "component", name, "error", err)
		}
	}

	for name := range installed {
		if _, required := manifestComponents[name]; !required {
			if _, err := fmt.Fprintf(s.Stdout, "  Note: Component %s is installed but not required by the template\n", name); err != nil {
				return fmt.Errorf("write message: %w", err)
			}
		}
	}

	if _, err := fmt.Fprintln(s.Stdout, strings.Repeat("-", 50)); err != nil {
		return fmt.Errorf("write separator: %w", err)
	}
	return nil
}
