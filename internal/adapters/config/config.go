package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/UNSAReport/UNSAReport/internal/ports"
)

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) FindProjectRoot(startDir string) (string, ports.UnsareportConfig, bool, error) {
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
	return startDir, ports.UnsareportConfig{}, false, nil
}

func (a *Adapter) ReadConfig(destDir string) (ports.UnsareportConfig, bool, error) {
	path := filepath.Join(destDir, "unsareport.json")
	var cfg ports.UnsareportConfig

	found := true
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			found = false
		} else {
			return ports.UnsareportConfig{}, false, fmt.Errorf("read file: %w", err)
		}
	}

	if found {
		if err := json.Unmarshal(b, &cfg); err != nil {
			return ports.UnsareportConfig{}, true, fmt.Errorf("failed to parse unsareport.json: %w", err)
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
	if cfg.Capture.Columns == 0 {
		cfg.Capture.Columns = 120
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

func (a *Adapter) WriteConfig(destDir string, cfg ports.UnsareportConfig) error {
	path := filepath.Join(destDir, "unsareport.json")

	// Set the schema version
	cfg.Schema = fmt.Sprintf("https://raw.githubusercontent.com/UNSAReport/UNSAReport/v%s/schemas/unsareport-v%s.schema.json", ports.Version, ports.Version)

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
