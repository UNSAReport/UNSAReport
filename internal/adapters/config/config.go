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
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, false, nil
		}
		return ports.LabReportConfig{}, false, fmt.Errorf("read file: %w", err)
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return ports.LabReportConfig{}, true, fmt.Errorf("failed to parse labreport.json: %w", err)
	}
	return cfg, true, nil
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
