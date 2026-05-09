package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/christianmz565/lab-report/internal/ports"
)

type CaptureService struct {
	Renderer ports.Renderer
	FS       ports.FileSystem
	Config   ports.ConfigStore
}

func NewCaptureService(r ports.Renderer, fs ports.FileSystem, c ports.ConfigStore) *CaptureService {
	return &CaptureService{
		Renderer: r,
		FS:       fs,
		Config:   c,
	}
}

func (s *CaptureService) Execute(ctx context.Context, tapeFile, cwdFlag string, args []string) error {
	cwd, err := s.FS.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}

	projectRoot, cfg, ok, err := s.Config.FindProjectRoot(cwd)
	if !ok {
		projectRoot = cwd
		cfg = ports.LabReportConfig{
			Prepare: ports.PrepareConfig{
				Input: ports.PrepareInputConfig{
					SrcDir:     "src",
					ReportFile: "report.typ",
				},
				Output: ports.PrepareOutputConfig{
					SubmissionDir: "submission",
				},
			},
			Capture: ports.CaptureConfig{
				TapeConfig: "config.tape",
			},
		}
	}

	if err := s.FS.Chdir(projectRoot); err != nil {
		return fmt.Errorf("chdir to project root: %w", err)
	}

	var tapePathToRun string

	if tapeFile != "" {
		tapePathAbs := filepath.Join(cwd, tapeFile)
		if !filepath.IsAbs(tapePathAbs) {
			var err error
			tapePathAbs, err = filepath.Abs(tapePathAbs)
			if err != nil {
				return fmt.Errorf("abs path: %w", err)
			}
		}
		tapePathToRun = tapePathAbs
	} else {
		if len(args) < 1 {
			return fmt.Errorf("result image path is required in oneshot mode")
		}
		resultPath := args[0]
		instructions := args[1:]

		var b strings.Builder

		if s.FS.FileExists(cfg.Capture.TapeConfig) {
			b.WriteString(fmt.Sprintf("Source %s\n\n", cfg.Capture.TapeConfig))
		}

		if cwdFlag != "" {
			b.WriteString(fmt.Sprintf("Type \"cd %s\"\nEnter\nType \"clear\"\nEnter\n\n", cwdFlag))
		}

		for _, instr := range instructions {
			if strings.HasPrefix(instr, "tape:") {
				b.WriteString(strings.TrimPrefix(instr, "tape:") + "\n")
			} else if strings.HasPrefix(instr, "\\tape:") {
				b.WriteString(fmt.Sprintf("Type \"%s\"\nEnter\nSleep 2\n", strings.ReplaceAll(instr[1:], "\"", "\\\"")))
			} else {
				b.WriteString(fmt.Sprintf("Type \"%s\"\nEnter\nSleep 2\n", strings.ReplaceAll(instr, "\"", "\\\"")))
			}
		}

		timestamp := time.Now().Format("02-01-2006_15-04-05")
		logPath := filepath.Join("capture_logs", timestamp+".ascii")

		if err := s.FS.EnsureDir("capture_logs"); err != nil {
			return fmt.Errorf("ensure capture_logs directory: %w", err)
		}

		b.WriteString(fmt.Sprintf("\nOutput %s\nScreenshot %s\nSleep 1\n", logPath, resultPath))

		tempFile, err := os.CreateTemp("", "lab-report-capture-*.tape")
		if err != nil {
			return fmt.Errorf("create temp tape file: %w", err)
		}
		defer os.Remove(tempFile.Name())

		if _, err := tempFile.WriteString(b.String()); err != nil {
			return fmt.Errorf("write temp tape file: %w", err)
		}
		if err := tempFile.Close(); err != nil {
			return fmt.Errorf("close temp tape file: %w", err)
		}

		tapePathToRun = tempFile.Name()
	}

	if err := s.Renderer.Render(ctx, tapePathToRun); err != nil {
		return fmt.Errorf("render tape: %w", err)
	}

	return nil
}
