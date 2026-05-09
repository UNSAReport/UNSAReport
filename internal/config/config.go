package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type SubmissionConfig struct {
	Template   string `json:"template,omitempty"`
	ReportWord string `json:"reportWord,omitempty"`
	CodeWord   string `json:"codeWord,omitempty"`
}

type LabReportConfig struct {
	MultiLab   bool             `json:"multiLab"`
	Submission SubmissionConfig `json:"submission"`
}

func FindProjectRoot(startDir string) (projectRoot string, cfg LabReportConfig, ok bool, err error) {
	currentDir := startDir
	for {
		cfg, ok, err = ReadConfig(currentDir)
		if ok || err != nil {
			return currentDir, cfg, ok, err
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}
	return startDir, DefaultConfig(), false, nil
}

func ReadConfig(destDir string) (cfg LabReportConfig, ok bool, err error) {
	path := filepath.Join(destDir, "labreport.json")
	cfg = DefaultConfig()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, false, nil
		}
		return LabReportConfig{}, false, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return LabReportConfig{}, true, fmt.Errorf("failed to parse labreport.json: %w", err)
	}
	return cfg, true, nil
}

func WriteConfig(destDir string, cfg LabReportConfig) error {
	path := filepath.Join(destDir, "labreport.json")
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func DefaultConfig() LabReportConfig {
	return LabReportConfig{}
}
