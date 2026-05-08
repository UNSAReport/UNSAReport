package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/christianmz565/lab-report/internal/config"
	"github.com/christianmz565/lab-report/internal/fsutil"
	"github.com/christianmz565/lab-report/internal/templates"
	"github.com/spf13/cobra"
)

type installOptions struct {
	dest  string
	multi bool
	repo  string
	ref   string
}

func newInstallCmd() *cobra.Command {
	opt := installOptions{}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the template into a destination directory",
		Long: `Install the lab report template files into a specified directory.
Downloads the latest templates from the repository and initializes the labreport.json configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmd.Help()
			}
			return runInstall(cmd.Context(), opt)
		},
	}

	cmd.Flags().StringVar(&opt.dest, "dest", "", "Destination directory (default: current working directory)")
	cmd.Flags().BoolVar(&opt.multi, "multi", false, "Install as a multi-lab repository")
	cmd.Flags().StringVar(&opt.repo, "repo", "christianmz565/lab-report", "GitHub repo to fetch templates from (owner/repo)")
	cmd.Flags().StringVar(&opt.ref, "ref", "main", "Git ref to fetch templates from")

	return cmd
}

func runInstall(ctx context.Context, opt installOptions) error {
	destDir := opt.dest
	if destDir == "" {
		destDir = "."
	}
	destDir, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}

	owner, repo, err := templates.ParseRepo(opt.repo)
	if err != nil {
		return err
	}
	src := templates.Source{Owner: owner, Repo: repo, Ref: opt.ref}

	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	files, err := templates.Fetch(ctx, src)
	if err != nil {
		// Dev/offline fallback: if running from the repo, use ./template.
		if local, lerr := templates.LoadFromDir("template"); lerr == nil {
			files = local
		} else {
			return err
		}
	}
	m, err := templates.LoadManifest(files)
	if err != nil {
		// Dev/offline fallback if the remote repo hasn't been updated yet.
		if local, lerr := templates.LoadFromDir("template"); lerr == nil {
			files = local
			m, err = templates.LoadManifest(files)
		}
		if err != nil {
			return err
		}
	}

	if err := fsutil.EnsureDir(destDir); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Installing lab report template to: %s\n", destDir)
	if opt.multi {
		fmt.Fprintln(os.Stdout, "Mode: Multi-lab (--multi)")
	}
	fmt.Fprintln(os.Stdout, strings.Repeat("-", 50))

	if opt.multi {
		if err := applyEntriesInstall(files, destDir, m.Multi.Root); err != nil {
			return err
		}
		if err := applyEntryInstall(files, destDir, m.Multi.Readme); err != nil {
			return err
		}

		lab := "l1"
		labEntries := substituteLab(m.Multi.LabFiles, lab)
		if err := applyEntriesInstall(files, destDir, labEntries); err != nil {
			return err
		}
	} else {
		if err := applyEntriesInstall(files, destDir, append(m.Common, m.Single...)); err != nil {
			return err
		}
	}

	if err := config.WriteConfig(destDir, config.LabReportConfig{MultiLab: opt.multi}); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Created: labreport.json (Mode: %s)\n", map[bool]string{true: "multi", false: "single"}[opt.multi])

	fmt.Fprintln(os.Stdout, strings.Repeat("-", 50))
	fmt.Fprintln(os.Stdout, "Installation complete!")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Next steps:")
	for _, s := range nextSteps(opt.multi) {
		fmt.Fprintln(os.Stdout, s)
	}

	return nil
}

func nextSteps(multi bool) []string {
	if multi {
		return []string{
			"1. Edit l1/report.typ with your lab information",
			"2. Place your code snippets in l1/snippets/",
			"3. Place your lab code in l1/src/",
			"4. Add screenshots to l1/img/lab/ as needed",
			"5. Run 'lab-report prepare l1' in the repo root when ready",
			"6. Duplicate l1 folder for subsequent labs (e.g., l2, l3)",
		}
	}
	return []string{
		"1. Edit report.typ with your lab information",
		"2. Place your code snippets in snippets/",
		"3. Place your lab code in src/",
		"4. Add screenshots to img/lab/ as needed",
		"5. Run 'lab-report prepare' when ready",
	}
}

func applyEntriesInstall(files templates.Files, destDir string, entries []templates.Entry) error {
	for _, e := range entries {
		if err := applyEntryInstall(files, destDir, e); err != nil {
			return err
		}
	}
	return nil
}

func applyEntryInstall(files templates.Files, destDir string, e templates.Entry) error {
	dstPath := filepath.Join(destDir, filepath.FromSlash(e.Dest))

	switch e.Kind {
	case templates.KindDir:
		if err := fsutil.EnsureDir(dstPath); err != nil {
			return err
		}
		// Copy any files under the directory if present in the remote template.
		srcPrefix := strings.TrimSuffix(e.Src, "/") + "/"
		for path, data := range files {
			if !strings.HasPrefix(path, srcPrefix) {
				continue
			}
			rel := strings.TrimPrefix(path, srcPrefix)
			if rel == "" {
				continue
			}
			dstFile := filepath.Join(dstPath, filepath.FromSlash(rel))
			if fsutil.FileExists(dstFile) {
				continue
			}
			if err := fsutil.WriteFileAtomic(dstFile, data, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Copied:  %s\n", filepath.ToSlash(filepath.Join(e.Dest, rel)))
		}
		return nil

	case templates.KindFile:
		data, ok := files[e.Src]
		if !ok {
			// For local-only files (like .prepare.config), create an empty file.
			if strings.HasPrefix(e.Src, ".") {
				if fsutil.FileExists(dstPath) {
					fmt.Fprintf(os.Stdout, "Skipped: %s (already exists)\n", e.Dest)
					return nil
				}
				if err := fsutil.WriteFileAtomic(dstPath, []byte(""), 0o644); err != nil {
					return err
				}
				fmt.Fprintf(os.Stdout, "Created: %s\n", e.Dest)
				return nil
			}
			return nil
		}

		if fsutil.FileExists(dstPath) {
			fmt.Fprintf(os.Stdout, "Skipped: %s (already exists)\n", e.Dest)
			return nil
		}
		if err := fsutil.WriteFileAtomic(dstPath, data, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Copied:  %s\n", e.Dest)
		return nil
	default:
		return fmt.Errorf("unknown manifest entry kind: %s", e.Kind)
	}
}

func substituteLab(entries []templates.Entry, lab string) []templates.Entry {
	out := make([]templates.Entry, 0, len(entries))
	for _, e := range entries {
		e2 := e
		e2.Dest = strings.ReplaceAll(e2.Dest, "{lab}", lab)
		out = append(out, e2)
	}
	return out
}

