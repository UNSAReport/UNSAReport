package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/christianmz565/lab-report/internal/config"
	"github.com/christianmz565/lab-report/internal/fsutil"
	"github.com/christianmz565/lab-report/internal/naming"
	"github.com/christianmz565/lab-report/internal/prepare"
	"github.com/spf13/cobra"
)

type prepareOptions struct {
	configure bool
}

type prepareContext struct {
	projectRoot string
	labDir      string
	isMulti     bool
	cfg         config.LabReportConfig
}

func newPrepareCmd() *cobra.Command {
	opt := prepareOptions{}
	cmd := &cobra.Command{
		Use:   "prepare [lab-dir]",
		Short: "Compile the Typst report and create the submission bundle",
		Long: `Compile the Typst report into a PDF and package the source code into a ZIP bundle.

In multi-lab mode, you must specify the lab directory (e.g., 'l1').`,
		Example: `  # Prepare submission in a single-lab project
  lab-report prepare

  # Prepare submission for a specific lab in multi-lab mode
  lab-report prepare l1

  # Reconfigure the output naming template
  lab-report prepare --configure`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var labDir string
			if len(args) == 1 {
				labDir = args[0]
			} else if len(args) > 1 {
				return cmd.Help()
			}
			return runPrepare(cmd.Context(), opt, labDir)
		},
	}

	cmd.Flags().BoolVar(&opt.configure, "configure", false, "Recreate the output-name template prompt")
	return cmd
}

func resolvePrepareContext(cwd, labDir string) (prepareContext, error) {
	pctx := prepareContext{labDir: labDir}
	projectRoot, cfg, ok, err := config.FindProjectRoot(cwd)
	if err != nil {
		fmt.Fprintln(os.Stdout, err.Error())
	}

	if ok {
		pctx.isMulti = cfg.MultiLab
	} else {
		defaultCfg := config.LabReportConfig{MultiLab: labDir != ""}
		if err := config.WriteConfig(cwd, defaultCfg); err != nil {
			return pctx, err
		}
		fmt.Fprintln(os.Stdout, "labreport.json not found. Created default config in the current directory.")
		fmt.Fprintln(os.Stdout, "Please validate the configuration and run the command again.")
		os.Exit(0)
	}

	if pctx.isMulti && pctx.labDir == "" {
		rel, err := filepath.Rel(projectRoot, cwd)
		if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			parts := strings.Split(rel, string(filepath.Separator))
			pctx.labDir = parts[0]
		}
		if pctx.labDir == "" {
			return pctx, fmt.Errorf("multi-lab mode detected. Please specify a lab directory (e.g., lab-report prepare l1)")
		}
	}

	pctx.projectRoot = projectRoot
	pctx.cfg = cfg
	return pctx, nil
}

func promptConfiguration(pctx *prepareContext, vars map[string]string) error {
	reportWord := pctx.cfg.Submission.ReportWord
	if reportWord == "" {
		reportWord = "Informe"
	}
	codeWord := pctx.cfg.Submission.CodeWord
	if codeWord == "" {
		codeWord = "Código Fuente"
	}

	input := pctx.cfg.Submission.Template
	if input == "" {
		input = "{outputType}_{lab_number}"
	}

	for {
		fmt.Fprintln(os.Stdout, "\nVariable configuration for report naming:")
		fmt.Fprintln(os.Stdout, "Available variables:")

		keys := make([]string, 0, len(vars))
		for k := range vars {
			keys = append(keys, k)
		}
		keys = append(keys, "outputType")
		sort.Strings(keys)
		for _, k := range keys {
			desc := vars[k]
			if k == "outputType" {
				desc = "Deliverable type (e.g., Informe or Código Fuente)"
			}
			fmt.Fprintf(os.Stdout, "  {%s}: %s\n", k, desc)
		}
		fmt.Fprintln(os.Stdout, "\nExample: {outputType}_{lab_number}_{members_abbr_list}")

		form := huh.NewForm(huh.NewGroup(
			huh.NewInput().
				Title("Enter the base name template (no extension)").
				Placeholder("{outputType}_{lab_number}").
				SuggestionsFunc(func() []string {
					suggestions := make([]string, 0)
					lastBrace := strings.LastIndex(input, "{")
					if lastBrace != -1 && lastBrace > strings.LastIndex(input, "}") {
						prefix := input[:lastBrace+1]
						for _, k := range keys {
							suggestions = append(suggestions, prefix+k+"}")
						}
					} else {
						suggestions = []string{"{outputType}_{lab_number}", "{outputType}_{lab_number}_{members_abbr_list}"}
					}
					return suggestions
				}, &input).
				Value(&input),
			huh.NewInput().
				Title("Word for the report file").
				Value(&reportWord),
			huh.NewInput().
				Title("Word for the source code file").
				Value(&codeWord),
		))
		if err := form.Run(); err != nil {
			return err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			input = "{outputType}_{lab_number}"
		}
		reportWord = strings.TrimSpace(reportWord)
		if reportWord == "" {
			reportWord = "Informe"
		}
		codeWord = strings.TrimSpace(codeWord)
		if codeWord == "" {
			codeWord = "Código Fuente"
		}

		previewReport := naming.ApplyTemplate(input, vars, reportWord)
		previewCode := naming.ApplyTemplate(input, vars, codeWord)

		var keep bool
		confirm := huh.NewForm(huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Preview: %s.pdf | %s.zip", previewReport, previewCode)).
				Value(&keep),
		))
		if err := confirm.Run(); err != nil {
			return err
		}
		if keep {
			pctx.cfg.Submission.Template = input
			pctx.cfg.Submission.ReportWord = reportWord
			pctx.cfg.Submission.CodeWord = codeWord
			if err := config.WriteConfig(pctx.projectRoot, pctx.cfg); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Configuration saved to labreport.json\n")
			break
		}
	}
	return nil
}

func runPrepare(ctx context.Context, opt prepareOptions, labDir string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	pctx, err := resolvePrepareContext(cwd, labDir)
	if err != nil {
		return err
	}

	if err := os.Chdir(pctx.projectRoot); err != nil {
		return err
	}

	reportPath := "report.typ"
	reportPDF := "report.pdf"
	srcDir := "src"
	if pctx.isMulti {
		reportPath = filepath.Join(pctx.labDir, "report.typ")
		reportPDF = filepath.Join(pctx.labDir, "report.pdf")
		srcDir = filepath.Join(pctx.labDir, "src")
	}

	if !fsutil.FileExists(reportPath) {
		return fmt.Errorf("report file not found: %s", reportPath)
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	vars, err := prepare.QueryVars(ctx, reportPath)
	if err != nil {
		return err
	}

	if pctx.cfg.Submission.Template == "" || opt.configure {
		if err := promptConfiguration(&pctx, vars); err != nil {
			return err
		}
	}

	reportWord := pctx.cfg.Submission.ReportWord
	if reportWord == "" {
		reportWord = "Informe"
	}
	codeWord := pctx.cfg.Submission.CodeWord
	if codeWord == "" {
		codeWord = "Código Fuente"
	}

	generatedReportName := naming.ApplyTemplate(pctx.cfg.Submission.Template, vars, reportWord)

	fmt.Fprintln(os.Stdout, "Compiling typst report...")
	if err := prepare.Compile(ctx, reportPath, reportPDF, generatedReportName); err != nil {
		return err
	}

	submissionDir := "submission"
	if pctx.isMulti {
		submissionDir = filepath.Join(pctx.labDir, "submission")
	}
	if err := fsutil.EnsureDir(submissionDir); err != nil {
		return err
	}

	reportFile := generatedReportName + ".pdf"
	codeFile := naming.ApplyTemplate(pctx.cfg.Submission.Template, vars, codeWord) + ".zip"

	if err := fsutil.CopyFile(reportPDF, filepath.Join(submissionDir, reportFile), 0o644); err != nil {
		return err
	}

	zipPath := filepath.Join(submissionDir, codeFile)
	_ = os.Remove(zipPath)

	fmt.Fprintf(os.Stdout, "Archiving %s to %s...\n", srcDir, zipPath)
	if err := prepare.ZipDir(zipPath, srcDir); err != nil {
		if err == prepare.ErrSourceMissing {
			fmt.Fprintf(os.Stdout, "Warning: %s directory not found. Skipping zip generation.\n", srcDir)
		} else {
			return err
		}
	}

	fmt.Fprintf(os.Stdout, "\nReport: %s\n", filepath.Join(submissionDir, reportFile))
	fmt.Fprintf(os.Stdout, "Code:  %s\n", filepath.Join(submissionDir, codeFile))
	return nil
}
