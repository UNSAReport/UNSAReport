package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/christianmz565/lab-report/internal/ports"
)

type PrepareOptions struct {
	Configure bool
}

type PrepareService struct {
	Compiler ports.Compiler
	Archiver ports.Archiver
	FS       ports.FileSystem
	Config   ports.ConfigStore
}

func NewPrepareService(c ports.Compiler, a ports.Archiver, fs ports.FileSystem, cfg ports.ConfigStore) *PrepareService {
	return &PrepareService{
		Compiler: c,
		Archiver: a,
		FS:       fs,
		Config:   cfg,
	}
}

type prepareContext struct {
	projectRoot string
	cfg         ports.LabReportConfig
	isMulti     bool
	labDir      string
}

func (s *PrepareService) Execute(ctx context.Context, opt PrepareOptions, labDirArg string) error {
	cwd, err := s.FS.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}

	pctx, err := s.resolvePrepareContext(cwd, labDirArg)
	if err != nil {
		return fmt.Errorf("resolve context: %w", err)
	}

	if err := s.FS.Chdir(pctx.projectRoot); err != nil {
		return fmt.Errorf("chdir to project root: %w", err)
	}

	reportPath := "report.typ"
	reportPDF := "report.pdf"
	srcDir := "src"
	if pctx.isMulti {
		reportPath = filepath.Join(pctx.labDir, "report.typ")
		reportPDF = filepath.Join(pctx.labDir, "report.pdf")
		srcDir = filepath.Join(pctx.labDir, "src")
	}

	if !s.FS.FileExists(reportPath) {
		return fmt.Errorf("report file not found: %s", reportPath)
	}

	vars, err := s.Compiler.QueryVars(ctx, reportPath)
	if err != nil {
		return fmt.Errorf("query vars: %w", err)
	}

	if pctx.cfg.Submission.Template == "" || opt.Configure {
		if err := s.promptConfiguration(&pctx, vars); err != nil {
			return fmt.Errorf("prompt config: %w", err)
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

	generatedReportName := ApplyTemplate(pctx.cfg.Submission.Template, vars, reportWord)

	fmt.Fprintln(os.Stdout, "Compiling typst report...")
	if err := s.Compiler.Compile(ctx, reportPath, reportPDF, generatedReportName); err != nil {
		return fmt.Errorf("compile report: %w", err)
	}

	submissionDir := "submission"
	if pctx.isMulti {
		submissionDir = filepath.Join(pctx.labDir, "submission")
	}
	if err := s.FS.EnsureDir(submissionDir); err != nil {
		return fmt.Errorf("ensure submission dir: %w", err)
	}

	reportFile := generatedReportName + ".pdf"
	codeFile := ApplyTemplate(pctx.cfg.Submission.Template, vars, codeWord) + ".zip"

	if err := s.FS.CopyFile(reportPDF, filepath.Join(submissionDir, reportFile), 0o644); err != nil {
		return fmt.Errorf("copy pdf: %w", err)
	}

	zipPath := filepath.Join(submissionDir, codeFile)
	_ = s.FS.Remove(zipPath)

	fmt.Fprintf(os.Stdout, "Archiving %s to %s...\n", srcDir, zipPath)
	if err := s.Archiver.ArchiveDir(zipPath, srcDir); err != nil {
		if strings.Contains(err.Error(), "source directory not found") {
			fmt.Fprintf(os.Stdout, "Warning: %s directory not found. Skipping zip generation.\n", srcDir)
		} else {
			return fmt.Errorf("archive dir: %w", err)
		}
	}

	fmt.Fprintf(os.Stdout, "\nReport: %s\n", filepath.Join(submissionDir, reportFile))
	fmt.Fprintf(os.Stdout, "Code:  %s\n", filepath.Join(submissionDir, codeFile))
	return nil
}

func (s *PrepareService) resolvePrepareContext(cwd, labDirArg string) (prepareContext, error) {
	projectRoot, cfg, ok, err := s.Config.FindProjectRoot(cwd)
	if err != nil {
		fmt.Fprintln(os.Stdout, err.Error())
	}

	if !ok {
		return prepareContext{}, fmt.Errorf("labreport.json not found in current or parent directories.\nAre you in a lab report project?")
	}

	pctx := prepareContext{
		projectRoot: projectRoot,
		cfg:         cfg,
		isMulti:     cfg.MultiLab,
	}

	if !pctx.isMulti {
		if labDirArg != "" {
			return pctx, fmt.Errorf("lab argument provided but project is not a multi-lab repository")
		}
		return pctx, nil
	}

	if labDirArg != "" {
		pctx.labDir = labDirArg
		return pctx, nil
	}

	rel, err := filepath.Rel(projectRoot, cwd)
	if err != nil {
		return pctx, fmt.Errorf("could not determine relative path: %w", err)
	}

	if rel == "." {
		return pctx, fmt.Errorf("in a multi-lab project, you must either provide a lab directory or run this command from inside a lab directory")
	}

	parts := strings.Split(filepath.ToSlash(rel), "/")
	pctx.labDir = parts[0]
	return pctx, nil
}

func (s *PrepareService) promptConfiguration(pctx *prepareContext, vars map[string]string) error {
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
		input = "{output_type}_{lab_number}"
	}

	for {
		fmt.Fprintln(os.Stdout, "\nVariable configuration for report naming:")
		fmt.Fprintln(os.Stdout, "Available variables:")

		keys := make([]string, 0, len(vars))
		for k := range vars {
			keys = append(keys, k)
		}
		keys = append(keys, "output_type")
		for _, k := range keys {
			desc := vars[k]
			if k == "output_type" {
				desc = "Deliverable type (e.g., Informe or Código Fuente)"
			}
			fmt.Fprintf(os.Stdout, "  {%s}: %s\n", k, desc)
		}
		fmt.Fprintln(os.Stdout, "\nExample: {output_type}_{lab_number}_{members_abbr_list}")

		form := huh.NewForm(huh.NewGroup(
			huh.NewInput().
				Title("Enter the base name template (no extension)").
				Placeholder("{output_type}_{lab_number}").
				SuggestionsFunc(func() []string {
					suggestions := make([]string, 0)
					lastBrace := strings.LastIndex(input, "{")
					if lastBrace != -1 && lastBrace > strings.LastIndex(input, "}") {
						prefix := input[:lastBrace+1]
						for _, k := range keys {
							suggestions = append(suggestions, prefix+k+"}")
						}
					} else {
						suggestions = []string{"{output_type}_{lab_number}", "{output_type}_{lab_number}_{members_abbr_list}"}
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
			input = "{output_type}_{lab_number}"
		}
		reportWord = strings.TrimSpace(reportWord)
		if reportWord == "" {
			reportWord = "Informe"
		}
		codeWord = strings.TrimSpace(codeWord)
		if codeWord == "" {
			codeWord = "Código Fuente"
		}

		previewReport := ApplyTemplate(input, vars, reportWord)
		previewCode := ApplyTemplate(input, vars, codeWord)

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
			if err := s.Config.WriteConfig(pctx.projectRoot, pctx.cfg); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Configuration saved to labreport.json\n")
			break
		}
	}
	return nil
}
