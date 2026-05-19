package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/christianmz565/lab-report/internal/ports"
)

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) FindProjectRoot(startDir string) (string, ports.LabReportConfig, bool, error) {
	currentDir := startDir
	for {
		cfg, ok, err := a.ReadConfig(currentDir)
		if ok || err != nil {
			return currentDir, cfg, ok, err
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}
	return startDir, ports.LabReportConfig{}, false, nil
}

func (a *Adapter) ReadConfig(destDir string) (ports.LabReportConfig, bool, error) {
	path := filepath.Join(destDir, "labreport.json")
	var cfg ports.LabReportConfig

	found := true
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			found = false
		} else {
			return ports.LabReportConfig{}, false, fmt.Errorf("read file: %w", err)
		}
	}

	if found {
		if err := json.Unmarshal(b, &cfg); err != nil {
			return ports.LabReportConfig{}, true, fmt.Errorf("failed to parse labreport.json: %w", err)
		}
	}

	if cfg.Prepare.Input.SrcDir == "" {
		cfg.Prepare.Input.SrcDir = "src"
	}
	if cfg.Prepare.Output.SubmissionDir == "" {
		cfg.Prepare.Output.SubmissionDir = "submission"
	}
	if cfg.Prepare.Input.ReportFile == "" {
		cfg.Prepare.Input.ReportFile = "report.typ"
	}
	if cfg.Capture.Prompt == "" {
		cfg.Capture.Prompt = "❯ "
	}
	if cfg.Capture.Colors == nil {
		cfg.Capture.Colors = map[string]string{
			"prompt":  "32",
			"command": "36",
			"args":    "33",
			"reset":   "0",
		}
	}
	return cfg, found, nil
}

func (a *Adapter) WriteConfig(destDir string, cfg ports.LabReportConfig) error {
	path := filepath.Join(destDir, "labreport.json")
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}
