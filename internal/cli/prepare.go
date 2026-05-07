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
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var labDir string
			if len(args) == 1 {
				labDir = args[0]
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
		return fmt.Errorf("labreport.json not found. Created default config; please verify and run again")
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

	templatePath := filepath.Join(cwd, ".prepare.config")
	template := ""
	if FileExists(templatePath) && !opt.configure {
		b, err := os.ReadFile(templatePath)
		if err == nil {
			template = strings.TrimSpace(string(b))
		}
	}

	for strings.TrimSpace(template) == "" {
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
		fmt.Fprintln(os.Stdout, "\nExample: Informe_LAB_{lab_number}_{members_abbr_list}")

		var input string
		form := huh.NewForm(huh.NewGroup(
			huh.NewInput().
				Title("Enter the name template (no extension)").
				Placeholder("Informe_LAB_{lab_number}").
				Value(&input),
		))
		if err := form.Run(); err != nil {
			return err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			input = "Informe_LAB_{lab_number}"
		}

		preview := applyTemplate(input, vars)
		var keep bool
		confirm := huh.NewForm(huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Preview: %s.pdf", preview)).
				Value(&keep),
		))
		if err := confirm.Run(); err != nil {
			return err
		}
		if keep {
			template = input
			if err := WriteFileAtomic(templatePath, []byte(template+"\n"), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Template saved to %s\n", templatePath)
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

	reportFile := generatedName + ".pdf"
	codeFile := generatedName + ".zip"

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
