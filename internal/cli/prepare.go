package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/christianmz565/lab-report/internal/prepare"
	"github.com/spf13/cobra"
)

type prepareOptions struct {
	configure bool
}

func newPrepareCmd() *cobra.Command {
	opt := prepareOptions{}
	cmd := &cobra.Command{
		Use:   "prepare [lab-dir]",
		Short: "Compile the Typst report and create the submission bundle",
		Long: `Compile the Typst report into PDF and package the source code into a ZIP for submission.

Arguments:
  [lab-dir]  Optional. Required only in multi-lab mode. Specifies the subdirectory for the specific lab (e.g., 'l1').`,
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

func runPrepare(ctx context.Context, opt prepareOptions, labDir string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	cfg, ok, err := ReadConfig(cwd)
	if err != nil {
		fmt.Fprintln(os.Stdout, err.Error())
	}

	isMulti := false
	if ok {
		isMulti = cfg.MultiLab
	} else {
		defaultCfg := LabReportConfig{MultiLab: labDir != ""}
		if err := WriteConfig(cwd, defaultCfg); err != nil {
			return err
		}
		cfg = defaultCfg
		fmt.Fprintln(os.Stdout, "labreport.json not found. Created default config.")
		isMulti = cfg.MultiLab
	}

	if isMulti && labDir == "" {
		return fmt.Errorf("multi-lab mode detected. Please specify a lab directory (e.g., lab-report prepare l1)")
	}

	reportPath := "report.typ"
	reportPDF := "report.pdf"
	srcDir := "src"
	if isMulti {
		reportPath = filepath.Join(labDir, "report.typ")
		reportPDF = filepath.Join(labDir, "report.pdf")
		srcDir = filepath.Join(labDir, "src")
	}

	if !FileExists(reportPath) {
		return fmt.Errorf("report file not found: %s", reportPath)
	}

	// Quick check for typst (also covered by global dependency check).
	_ = exec.Command("typst", "--version").Run()

	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	vars, err := prepare.QueryVars(ctx, reportPath, isMulti)
	if err != nil {
		return err
	}

	template := ""
	if !opt.configure && cfg.SubmissionTemplate != "" {
		template = cfg.SubmissionTemplate
	}

	reportWord := cfg.ReportWord
	if reportWord == "" {
		reportWord = "Informe"
	}
	codeWord := cfg.CodeWord
	if codeWord == "" {
		codeWord = "Código Fuente"
	}

	input := template
	if input == "" {
		input = "LAB_{lab_number}"
	}

	for template == "" || opt.configure {
		fmt.Fprintln(os.Stdout, "\nVariable configuration for report naming:")
		fmt.Fprintln(os.Stdout, "Available variables:")

		keys := make([]string, 0, len(vars))
		for k := range vars {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(os.Stdout, "  {%s}: %s\n", k, vars[k])
		}
		fmt.Fprintln(os.Stdout, "\nExample: LAB_{lab_number}_{members_abbr_list}")

		form := huh.NewForm(huh.NewGroup(
			huh.NewInput().
				Title("Enter the base name template (no extension)").
				Placeholder("LAB_{lab_number}").
				Suggestions([]string{"LAB_{lab_number}", "LAB_{lab_number}_{members_abbr_list}"}).
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
			input = "LAB_{lab_number}"
		}
		reportWord = strings.TrimSpace(reportWord)
		if reportWord == "" {
			reportWord = "Informe"
		}
		codeWord = strings.TrimSpace(codeWord)
		if codeWord == "" {
			codeWord = "Código Fuente"
		}

		previewName := applyTemplate(input, vars)
		var keep bool
		confirm := huh.NewForm(huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Preview: %s_%s.pdf | %s_%s.zip", reportWord, previewName, codeWord, previewName)).
				Value(&keep),
		))
		if err := confirm.Run(); err != nil {
			return err
		}
		if keep {
			template = input
			cfg.SubmissionTemplate = template
			cfg.ReportWord = reportWord
			cfg.CodeWord = codeWord
			if err := WriteConfig(cwd, cfg); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Configuration saved to labreport.json\n")
			opt.configure = false // Break the loop
		}
	}

	generatedName := applyTemplate(template, vars)

	fmt.Fprintln(os.Stdout, "Compiling typst report...")
	compileArgs := []string{"compile"}
	if isMulti {
		compileArgs = append(compileArgs, "--root", ".")
	}
	compileArgs = append(compileArgs, "--input", fmt.Sprintf("title=%s", generatedName), reportPath, reportPDF)

	compileCmd := exec.CommandContext(ctx, "typst", compileArgs...)
	compileCmd.Stdout = os.Stdout
	compileCmd.Stderr = os.Stderr
	if err := compileCmd.Run(); err != nil {
		return err
	}

	submissionDir := "submission"
	if isMulti {
		submissionDir = filepath.Join(labDir, "submission")
	}
	if err := EnsureDir(submissionDir); err != nil {
		return err
	}

	reportFile := fmt.Sprintf("%s_%s.pdf", cfg.ReportWord, generatedName)
	codeFile := fmt.Sprintf("%s_%s.zip", cfg.CodeWord, generatedName)

	if err := CopyFile(reportPDF, filepath.Join(submissionDir, reportFile), 0o644); err != nil {
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

var reVar = regexp.MustCompile(`\{(\w+)\}`)

func applyTemplate(tpl string, vars map[string]string) string {
	return reVar.ReplaceAllStringFunc(tpl, func(m string) string {
		sub := reVar.FindStringSubmatch(m)
		if len(sub) == 2 {
			if v, ok := vars[sub[1]]; ok {
				return v
			}
		}
		return m
	})
}
