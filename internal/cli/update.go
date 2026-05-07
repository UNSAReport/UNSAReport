package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/christianmz565/lab-report/internal/diff"
	"github.com/christianmz565/lab-report/internal/templates"
	"github.com/spf13/cobra"
)

type updateOptions struct {
	dest  string
	force bool
	multi bool
	repo  string
	ref   string
}

func newUpdateCmd() *cobra.Command {
	opt := updateOptions{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an existing template installation",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd.Context(), opt)
		},
	}

	cmd.Flags().StringVar(&opt.dest, "dest", "", "Destination directory (default: current working directory)")
	cmd.Flags().BoolVarP(&opt.force, "force", "f", false, "Apply all updates without prompting")
	cmd.Flags().BoolVar(&opt.multi, "multi", false, "Force multi-lab update mode")
	cmd.Flags().StringVar(&opt.repo, "repo", "christianmz565/lab-report", "GitHub repo to fetch templates from (owner/repo)")
	cmd.Flags().StringVar(&opt.ref, "ref", "main", "Git ref to fetch templates from")

	return cmd
}

type updateDecision string

const (
	decYes  updateDecision = "yes"
	decNo   updateDecision = "no"
	decAll  updateDecision = "all"
	decQuit updateDecision = "quit"
)

var imageExt = regexp.MustCompile(`(?i)\.(png|jpe?g|gif|svg|webp|ico)$`)

func runUpdate(ctx context.Context, opt updateOptions) error {
	destDir := opt.dest
	if destDir == "" {
		destDir = "."
	}
	destDir, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}

	cfg, ok, err := ReadConfig(destDir)
	if err != nil {
		fmt.Fprintln(os.Stdout, err.Error())
		// keep going; we'll fall back to flags
	}

	isMulti := opt.multi
	if ok {
		isMulti = isMulti || cfg.MultiLab
	} else {
		// Mimic old behavior: create default config and ask user to re-run.
		defaultCfg := LabReportConfig{MultiLab: isMulti}
		if err := WriteConfig(destDir, defaultCfg); err != nil {
			return err
		}
		return fmt.Errorf("labreport.json not found. Created default config; please verify and run update again")
	}

	owner, repo, err := templates.ParseRepo(opt.repo)
	if err != nil {
		return err
	}
	src := templates.Source{Owner: owner, Repo: repo, Ref: opt.ref}

	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	remoteFiles, err := templates.Fetch(ctx, src)
	if err != nil {
		// Dev/offline fallback: if running from the repo, use ./template.
		if local, lerr := templates.LoadFromDir("template"); lerr == nil {
			remoteFiles = local
		} else {
			return err
		}
	}
	m, err := templates.LoadManifest(remoteFiles)
	if err != nil {
		// Dev/offline fallback if the remote repo hasn't been updated yet.
		if local, lerr := templates.LoadFromDir("template"); lerr == nil {
			remoteFiles = local
			m, err = templates.LoadManifest(remoteFiles)
		}
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stdout, "Detected %s setup.\n", map[bool]string{true: "multi-lab", false: "single-lab"}[isMulti])
	fmt.Fprintf(os.Stdout, "Checking for updates in: %s\n\n", destDir)

	if err := maybeRemoveLegacyScripts(destDir, opt.force); err != nil {
		return err
	}

	// .prepare.config is user-owned; ensure it exists but never update it from remote templates.
	prepareCfg := filepath.Join(destDir, ".prepare.config")
	if !FileExists(prepareCfg) {
		if err := WriteFileAtomic(prepareCfg, []byte(""), 0o644); err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, "Created: .prepare.config")
	}

	entries := buildUpdateEntries(m, isMulti, destDir)
	entries = expandDirEntries(remoteFiles, entries)
	applied := 0
	autoAcceptAll := opt.force

	for _, e := range entries {
		dstPath := filepath.Join(destDir, filepath.FromSlash(e.Dest))
		srcPath := e.Src

		switch e.Kind {
		case templates.KindDir:
			if err := EnsureDir(dstPath); err != nil {
				return err
			}
			continue
		case templates.KindFile:
			remote, ok := remoteFiles[srcPath]
			if !ok {
				continue
			}

			local, lerr := os.ReadFile(dstPath)
			isNew := os.IsNotExist(lerr)
			if lerr != nil && !isNew {
				return lerr
			}
			if !isNew && SameContent(local, remote) {
				continue
			}

			label := e.Dest
			apply := func() error {
				if err := WriteFileAtomic(dstPath, remote, 0o644); err != nil {
					return err
				}
				applied++
				return nil
			}

			if e.AutoUpdate || autoAcceptAll {
				if err := apply(); err != nil {
					return err
				}
				tag := "[FORCED]"
				if e.AutoUpdate {
					tag = "[AUTO-UPDATE]"
				}
				fmt.Fprintf(os.Stdout, "✓ Updated %s: %s\n", tag, label)
				continue
			}

			if isNew {
				fmt.Fprintf(os.Stdout, "\n[NEW FILE] %s\n", label)
			} else {
				showDiff(label, string(local), string(remote), dstPath, srcPath)
			}

			dec, err := promptUpdateDecision(label)
			if err != nil {
				return err
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

func buildUpdateEntries(m *templates.Manifest, isMulti bool, destDir string) []templates.Entry {
	var out []templates.Entry
	add := func(entries ...[]templates.Entry) {
		for _, list := range entries {
			out = append(out, list...)
		}
	}

	if isMulti {
		add(m.Multi.Root)
		out = append(out, m.Multi.Readme)

		labs := detectLabDirs(destDir)
		sort.Strings(labs)
		for _, lab := range labs {
			out = append(out, substituteLab(m.Multi.LabFiles, lab)...)
		}
	} else {
		add(append(m.Common, m.Single...))
	}

	// Dedup by destination path (last write wins).
	seen := map[string]templates.Entry{}
	for _, e := range out {
		seen[e.Dest] = e
	}

	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	final := make([]templates.Entry, 0, len(keys))
	for _, k := range keys {
		final = append(final, seen[k])
	}
	return final
}

func expandDirEntries(remote templates.Files, entries []templates.Entry) []templates.Entry {
	out := make([]templates.Entry, 0, len(entries))
	for _, e := range entries {
		out = append(out, e)
		if e.Kind != templates.KindDir || strings.TrimSpace(e.Src) == "" {
			continue
		}

		prefix := strings.TrimSuffix(e.Src, "/") + "/"
		paths := make([]string, 0)
		for p := range remote {
			if strings.HasPrefix(p, prefix) {
				paths = append(paths, p)
			}
		}
		sort.Strings(paths)

		for _, p := range paths {
			rel := strings.TrimPrefix(p, prefix)
			if rel == "" {
				continue
			}
			out = append(out, templates.Entry{
				Kind: templates.KindFile,
				Src:  p,
				Dest: filepath.ToSlash(filepath.Join(e.Dest, rel)),
			})
		}
	}
	return out
}

func detectLabDirs(destDir string) []string {
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return nil
	}
	labs := make([]string, 0)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" {
			continue
		}
		if FileExists(filepath.Join(destDir, name, "report.typ")) {
			labs = append(labs, name)
		}
	}
	return labs
}

func showDiff(fileName, oldText, newText, oldPath, newSrc string) {
	fmt.Fprintf(os.Stdout, "\n--- Diff for %s ---\n", fileName)
	if imageExt.MatchString(fileName) {
		fmt.Fprintln(os.Stdout, "  [Binary image file changed. Cannot display inline diff.]")
		fmt.Fprintf(os.Stdout, "  - Local version at: %s\n", oldPath)
		fmt.Fprintf(os.Stdout, "  + New template at: %s\n", newSrc)
		fmt.Fprintln(os.Stdout, strings.Repeat("-", 30)+"\n")
		return
	}

	fmt.Fprint(os.Stdout, diff.UnifiedLineDiff(oldText, newText))
	fmt.Fprintln(os.Stdout, strings.Repeat("-", 30)+"\n")
}

func promptUpdateDecision(label string) (updateDecision, error) {
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

func maybeRemoveLegacyScripts(destDir string, force bool) error {
	legacy := []string{"prepare.js", "capture.js"}
	found := make([]string, 0)
	for _, f := range legacy {
		p := filepath.Join(destDir, f)
		if FileExists(p) {
			found = append(found, f)
		}
	}
	if len(found) == 0 {
		return nil
	}

	if force {
		for _, f := range found {
			_ = os.Remove(filepath.Join(destDir, f))
			fmt.Fprintf(os.Stdout, "Removed legacy script: %s\n", f)
		}
		return nil
	}

	var confirm bool
	form := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title("Remove legacy Node scripts (prepare.js/capture.js)?").
			Description("These are no longer needed; use `lab-report prepare` and `lab-report capture` instead.").
			Value(&confirm),
	))
	if err := form.Run(); err != nil {
		return err
	}
	if !confirm {
		return nil
	}

	for _, f := range found {
		if err := os.Remove(filepath.Join(destDir, f)); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Removed legacy script: %s\n", f)
	}
	return nil
}
