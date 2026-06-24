package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/UNSAReport/UNSAReport/internal/ports"
)

func TestReadConfig_Defaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "unsareport.json")
	err := os.WriteFile(cfgPath, []byte(`{}`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	a := &Adapter{}
	cfg, found, err := a.ReadConfig(dir)
	if err != nil {
		t.Fatalf("ReadConfig() error = %v", err)
	}

	if !found {
		t.Fatal("expected found = true")
	}

	if cfg.Prepare.Input.SrcDir != "src" {
		t.Errorf("SrcDir = %q, want %q", cfg.Prepare.Input.SrcDir, "src")
	}

	if cfg.Prepare.Output.SubmissionDir != "submission" {
		t.Errorf("SubmissionDir = %q, want %q", cfg.Prepare.Output.SubmissionDir, "submission")
	}

	if cfg.Prepare.Input.ReportFile != "report.typ" {
		t.Errorf("ReportFile = %q, want %q", cfg.Prepare.Input.ReportFile, "report.typ")
	}

	if cfg.Capture.Prompt != "❯ " {
		t.Errorf("Prompt = %q, want %q", cfg.Capture.Prompt, "❯ ")
	}

	if cfg.Capture.Columns != 120 {
		t.Errorf("Columns = %d, want 120", cfg.Capture.Columns)
	}

	if cfg.Capture.Rows != 500 {
		t.Errorf("Rows = %d, want 500", cfg.Capture.Rows)
	}
}

func TestReadConfig_InvalidMode(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "unsareport.json")
	err := os.WriteFile(cfgPath, []byte(`{"mode": "invalid"}`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	a := &Adapter{}
	_, _, err = a.ReadConfig(dir)
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestReadConfig_MultiNoSessions(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "unsareport.json")
	err := os.WriteFile(cfgPath, []byte(`{"mode": "multi", "sessions": []}`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	a := &Adapter{}
	_, _, err = a.ReadConfig(dir)
	if err == nil {
		t.Fatal("expected error for multi-mode with no sessions")
	}
}

func TestReadConfig_NotFound(t *testing.T) {
	dir := t.TempDir()
	a := &Adapter{}
	_, found, err := a.ReadConfig(dir)
	if err != nil {
		t.Fatalf("ReadConfig() error = %v", err)
	}

	if found {
		t.Fatal("expected found = false")
	}
}

func TestWriteConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	a := &Adapter{}

	cfg := ports.UnsareportConfig{
		Template: "lab",
		Mode:     "single",
		Sessions: []string{},
		Capture: ports.CaptureConfig{
			Columns: 120,
			Rows:    500,
			Prompt:  "❯ ",
			Colors:  map[string]string{"prompt": "32"},
		},
	}

	if err := a.WriteConfig(dir, cfg); err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
	}

	got, found, err := a.ReadConfig(dir)
	if err != nil {
		t.Fatalf("ReadConfig() error = %v", err)
	}

	if !found {
		t.Fatal("expected found = true after write")
	}

	if got.Template != "lab" {
		t.Errorf("Template = %q, want %q", got.Template, "lab")
	}

	if got.Mode != "single" {
		t.Errorf("Mode = %q, want %q", got.Mode, "single")
	}
}

func TestComputeIntegrity(t *testing.T) {
	data := []byte("hello world")
	integrity := ComputeIntegrity(data)

	if integrity == "" {
		t.Fatal("expected non-empty integrity")
	}

	if len(integrity) != 71 {
		t.Errorf("integrity length = %d, want 71", len(integrity))
	}

	integrity2 := ComputeIntegrity(data)
	if integrity != integrity2 {
		t.Errorf("same data produced different integrity: %q vs %q", integrity, integrity2)
	}
}
