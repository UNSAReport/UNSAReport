package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/UNSAReport/UNSAReport/internal/ports"
	"github.com/charmbracelet/huh"
)

type UpdateOptions struct {
	Dest    string
	Force   bool
	Session string
	Repo    string
	Ref     string
}

type UpdateService struct {
	Fetcher  ports.TemplateFetcher
	FS       ports.FileSystem
	Config   ports.ConfigStore
	Registry ports.TemplateRegistry
}

func NewUpdateService(f ports.TemplateFetcher, fs ports.FileSystem, c ports.ConfigStore, r ports.TemplateRegistry) *UpdateService {
	return &UpdateService{
		Fetcher:  f,
		FS:       fs,
		Config:   c,
		Registry: r,
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
		fmt.Fprintln(os.Stdout, err.Error())
	}

	isMulti := false
	if ok {
		isMulti = cfg.MultiLab
		destDir = projectRoot
	} else {
		defaultCfg := ports.UnsareportConfig{
			MultiLab: isMulti,
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
		if err := s.Config.WriteConfig(destDir, defaultCfg); err != nil {
			return fmt.Errorf("write default config: %w", err)
		}
		fmt.Fprintln(os.Stdout, "unsareport.json not found. Created default config in the target directory.")
		fmt.Fprintln(os.Stdout, "Please validate the configuration and run the command again.")
		return nil
	}

	if err := s.FS.Chdir(destDir); err != nil {
		return fmt.Errorf("chdir to dest: %w", err)
	}

	var remoteFiles map[string][]byte
	if info, err := s.FS.Stat(opt.Repo); err == nil && info.IsDir() {
		remoteFiles, err = s.Fetcher.LoadLocal(opt.Repo)
		if err != nil {
			return fmt.Errorf("load local templates: %w", err)
		}
	} else {
		remoteFiles, err = s.Fetcher.Fetch(ctx, opt.Repo, opt.Ref)
		if err != nil {
			if local, lerr := s.Fetcher.LoadLocal("template"); lerr == nil {
				remoteFiles = local
			} else {
				return fmt.Errorf("fetch templates: %w", err)
			}
		}
	}

	m, err := LoadManifest(remoteFiles)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Detected %s setup.\n", map[bool]string{true: "multi-lab", false: "single-lab"}[isMulti])
	fmt.Fprintf(os.Stdout, "Checking for updates in: %s\n\n", destDir)

	entries := s.buildUpdateEntries(m, isMulti, destDir, cfg, opt.Session)
	entries = ExpandDirEntries(remoteFiles, entries)
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
				fmt.Fprintf(os.Stdout, "✓ Updated [FORCED]: %s\n", label)
				continue
			}

			if isNew {
				fmt.Fprintf(os.Stdout, "\n[NEW FILE] %s\n", label)
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
				fmt.Fprintf(os.Stdout, "✓ Updated: %s\n", label)
			case decNo:
				fmt.Fprintf(os.Stdout, "Skipped: %s\n", label)
			case decQuit:
				fmt.Fprintln(os.Stdout, "\nStopping update process.")
				return nil
			}
		default:
			return fmt.Errorf("unknown manifest entry kind: %s", e.Kind)
		}
	}

	fmt.Fprintf(os.Stdout, "\nUpdate finished. %d files updated.\n", applied)
	return nil
}

func (s *UpdateService) buildUpdateEntries(m *Manifest, isMulti bool, destDir string, cfg ports.UnsareportConfig, session string) []Entry {
	var out []Entry
	add := func(entries ...[]Entry) {
		for _, list := range entries {
			out = append(out, list...)
		}
	}

	if isMulti {
		add(m.Common, m.Multi.Root)

		labs := cfg.Sessions
		if session != "" {
			labs = []string{session}
		}

		for _, lab := range labs {
			out = append(out, substituteLab(m.Multi.LabFiles, lab)...)
		}
	} else {
		add(append(m.Common, m.Single...))
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
	fmt.Fprintf(os.Stdout, "\n--- Diff for %s ---\n", fileName)
	if imageExt.MatchString(fileName) {
		fmt.Fprintln(os.Stdout, "  [Binary image file changed. Cannot display inline diff.]")
		fmt.Fprintf(os.Stdout, "  - Local version at: %s\n", oldPath)
		fmt.Fprintf(os.Stdout, "  + New template at: %s\n", newSrc)
		fmt.Fprintln(os.Stdout, strings.Repeat("-", 30)+"\n")
		return
	}

	fmt.Fprint(os.Stdout, UnifiedLineDiff(oldText, newText))
	fmt.Fprintln(os.Stdout, strings.Repeat("-", 30)+"\n")
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
