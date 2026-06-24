package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/UNSAReport/UNSAReport/internal/adapters/zipper"
	"github.com/UNSAReport/UNSAReport/internal/ports"
	"github.com/charmbracelet/huh"
)

type PrepareOptions struct {
	Configure bool
}

type PrepareService struct {
	Compiler ports.Compiler
	Archiver ports.Archiver
	FS       ports.FileSystem
	Config   ports.ConfigStore
	Stdout   io.Writer
	Stderr   io.Writer
}

func NewPrepareService(c ports.Compiler, a ports.Archiver, fs ports.FileSystem, cfg ports.ConfigStore, stdout, stderr io.Writer) *PrepareService {
	return &PrepareService{
		Compiler: c,
		Archiver: a,
		FS:       fs,
		Config:   cfg,
		Stdout:   stdout,
		Stderr:   stderr,
	}
}

type prepareContext struct {
	projectRoot string
	cfg         ports.UnsareportConfig
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

	reportPath := pctx.cfg.Prepare.Input.ReportFile
	reportPDF := strings.TrimSuffix(pctx.cfg.Prepare.Input.ReportFile, filepath.Ext(pctx.cfg.Prepare.Input.ReportFile)) + ".pdf"
	srcDir := pctx.cfg.Prepare.Input.SrcDir
	if pctx.isMulti {
		reportPath = filepath.Join(pctx.labDir, pctx.cfg.Prepare.Input.ReportFile)
		reportPDF = filepath.Join(pctx.labDir, strings.TrimSuffix(pctx.cfg.Prepare.Input.ReportFile, filepath.Ext(pctx.cfg.Prepare.Input.ReportFile))+".pdf")
		srcDir = filepath.Join(pctx.labDir, pctx.cfg.Prepare.Input.SrcDir)
	}

	if !s.FS.FileExists(reportPath) {
		return fmt.Errorf("report file not found: %s", reportPath)
	}

	vars, err := s.Compiler.QueryVars(ctx, reportPath)
	if err != nil {
		return fmt.Errorf("query vars: %w", err)
	}

	if pctx.cfg.Prepare.Output.FileTemplate == "" || opt.Configure {
		if err := s.promptConfiguration(&pctx, vars); err != nil {
			return fmt.Errorf("prompt config: %w", err)
		}
	}

	reportWord := pctx.cfg.Prepare.Output.ReportWord
	if reportWord == "" {
		reportWord = "Informe"
	}
	codeWord := pctx.cfg.Prepare.Output.CodeWord
	if codeWord == "" {
		codeWord = "Código Fuente"
	}

	generatedReportName := ApplyTemplate(pctx.cfg.Prepare.Output.FileTemplate, vars, reportWord)

	fmt.Fprintln(s.Stdout, "Compiling typst report...")
	inputs := map[string]string{"title": generatedReportName}
	if pctx.isMulti {
		inputs["unsarep-root"] = "/" + pctx.labDir + "/"
	} else {
		inputs["unsarep-root"] = "/"
	}
	if err := s.Compiler.Compile(ctx, reportPath, reportPDF, inputs); err != nil {
		return fmt.Errorf("compile report: %w", err)
	}

	submissionDir := pctx.cfg.Prepare.Output.SubmissionDir
	if pctx.isMulti {
		submissionDir = filepath.Join(pctx.labDir, pctx.cfg.Prepare.Output.SubmissionDir)
	}
	if err := s.FS.EnsureDir(submissionDir); err != nil {
		return fmt.Errorf("ensure submission dir: %w", err)
	}

	reportFile := generatedReportName + ".pdf"
	codeFile := ApplyTemplate(pctx.cfg.Prepare.Output.FileTemplate, vars, codeWord) + ".zip"

	if err := s.FS.CopyFile(reportPDF, filepath.Join(submissionDir, reportFile), 0o644); err != nil {
		return fmt.Errorf("copy pdf: %w", err)
	}

	zipPath := filepath.Join(submissionDir, codeFile)
	if err := s.FS.Remove(zipPath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(s.Stderr, "Warning: could not remove old zip: %v\n", err)
	}

	fmt.Fprintf(s.Stdout, "Archiving %s to %s...\n", srcDir, zipPath)
	files, err := s.listGitFiles(ctx, srcDir)
	if err != nil {
		return fmt.Errorf("list git files: %w", err)
	}

	if files != nil {
		if err := s.Archiver.ArchiveFiles(zipPath, srcDir, files); err != nil {
			return fmt.Errorf("archive files: %w", err)
		}
	} else {
		if err := s.Archiver.ArchiveDir(zipPath, srcDir); err != nil {
			if errors.Is(err, zipper.ErrSourceMissing) {
				fmt.Fprintf(s.Stderr, "Warning: %s directory not found. Skipping zip generation.\n", srcDir)
			} else {
				return fmt.Errorf("archive dir: %w", err)
			}
		}
	}

	fmt.Fprintf(s.Stdout, "\nReport: %s\n", filepath.Join(submissionDir, reportFile))
	fmt.Fprintf(s.Stdout, "Code:  %s\n", filepath.Join(submissionDir, codeFile))
	return nil
}

func (s *PrepareService) listGitFiles(ctx context.Context, srcDir string) ([]string, error) {
	if !s.FS.FileExists(".git") {
		return nil, nil
	}

	if _, err := exec.LookPath("git"); err != nil {
		return nil, nil
	}

	cmd := exec.CommandContext(ctx, "git", "ls-files", "--cached", "--others", "--exclude-standard", srcDir)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var files []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		rel, err := filepath.Rel(srcDir, line)
		if err != nil {
			continue
		}
		files = append(files, rel)
	}

	if len(files) == 0 && !s.FS.FileExists(srcDir) {
		return nil, fmt.Errorf("source directory not found")
	}

	return files, nil
}

func (s *PrepareService) resolvePrepareContext(cwd, labDirArg string) (prepareContext, error) {
	projectRoot, cfg, ok, err := s.Config.FindProjectRoot(cwd)
	if err != nil {
		return prepareContext{}, fmt.Errorf("find project root: %w", err)
	}

	if !ok {
		return prepareContext{}, fmt.Errorf("unsareport.json not found in current or parent directories.\nAre you in a lab report project?")
	}

	pctx := prepareContext{
		projectRoot: projectRoot,
		cfg:         cfg,
		isMulti:     cfg.Mode == "multi",
	}

	if !pctx.isMulti {
		if labDirArg != "" {
			return pctx, fmt.Errorf("lab argument provided but template is not multi-mode")
		}
		return pctx, nil
	}

	if labDirArg != "" {
		pctx.labDir = labDirArg
	} else {
		rel, err := filepath.Rel(projectRoot, cwd)
		if err != nil {
			return pctx, fmt.Errorf("could not determine relative path: %w", err)
		}

		if rel == "." {
			return pctx, fmt.Errorf("in a multi-mode project, you must either provide a lab directory or run this command from inside a lab directory")
		}

		parts := strings.Split(filepath.ToSlash(rel), "/")
		pctx.labDir = parts[0]
	}

	sessionValid := slices.Contains(pctx.cfg.Sessions, pctx.labDir)
	if !sessionValid {
		return pctx, fmt.Errorf("session '%s' is not registered in unsareport.json", pctx.labDir)
	}

	return pctx, nil
}

func (s *PrepareService) promptConfiguration(pctx *prepareContext, vars map[string]string) error {
	reportWord := pctx.cfg.Prepare.Output.ReportWord
	if reportWord == "" {
		reportWord = "Informe"
	}
	codeWord := pctx.cfg.Prepare.Output.CodeWord
	if codeWord == "" {
		codeWord = "Código Fuente"
	}

	input := pctx.cfg.Prepare.Output.FileTemplate
	if input == "" {
		input = "{output_type}_{lab_number}"
	}

	for {
		fmt.Fprintln(s.Stdout, "\nVariable configuration for report naming:")
		fmt.Fprintln(s.Stdout, "Available variables:")

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
			fmt.Fprintf(s.Stdout, "  {%s}: %s\n", k, desc)
		}
		fmt.Fprintln(s.Stdout, "\nExample: {output_type}_{lab_number}_{members_abbr_list}")

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
			pctx.cfg.Prepare.Output.FileTemplate = input
			pctx.cfg.Prepare.Output.ReportWord = reportWord
			pctx.cfg.Prepare.Output.CodeWord = codeWord
			if err := s.Config.WriteConfig(pctx.projectRoot, pctx.cfg); err != nil {
				return err
			}
			fmt.Fprintf(s.Stdout, "Configuration saved to unsareport.json\n")
			break
		}
	}
	return nil
}
