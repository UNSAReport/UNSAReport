package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type LabReportConfig struct {
	MultiLab           bool              `json:"multiLab"`
	SubmissionTemplate string            `json:"submissionTemplate,omitempty"`
	ReportWord         string            `json:"reportWord,omitempty"`
	CodeWord           string            `json:"codeWord,omitempty"`
	FreezeFlags        []string          `json:"freezeFlags,omitempty"`
	CapturePrompt      string            `json:"capturePrompt,omitempty"`
	CaptureColors      map[string]string `json:"captureColors,omitempty"`
}

func ReadConfig(destDir string) (cfg LabReportConfig, ok bool, err error) {
	path := filepath.Join(destDir, "labreport.json")
	cfg = DefaultConfig() // Start with defaults
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
	return LabReportConfig{
		CapturePrompt: "❯ ",
		CaptureColors: map[string]string{
			"prompt":   "38;5;114",
			"command":  "38;5;111",
			"args":     "38;5;217",
			"reset":    "0",
		},
	}
}
