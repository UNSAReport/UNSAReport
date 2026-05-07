package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type LabReportConfig struct {
	MultiLab bool `json:"multiLab"`
}

func ReadConfig(destDir string) (cfg LabReportConfig, ok bool, err error) {
	path := filepath.Join(destDir, "labreport.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return LabReportConfig{}, false, nil
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
