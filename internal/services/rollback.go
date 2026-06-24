package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/UNSAReport/UNSAReport/internal/ports"
)

const backupDir = ".unsarep-backup"

type BackupManifest struct {
	Timestamp       string                `json:"timestamp"`
	TemplateVersion string                `json:"template_version"`
	Files           []BackupManifestEntry `json:"files"`
}

type BackupManifestEntry struct {
	RelativePath string `json:"relative_path"`
	OriginalPath string `json:"original_path"`
}

type RollbackService struct {
	FS     ports.FileSystem
	Config ports.ConfigStore
	Stdout io.Writer
	Stderr io.Writer
}

func NewRollbackService(fs ports.FileSystem, cfg ports.ConfigStore, stdout, stderr io.Writer) *RollbackService {
	return &RollbackService{FS: fs, Config: cfg, Stdout: stdout, Stderr: stderr}
}

func (s *RollbackService) CreateBackup(destDir string, entries []Entry, cfg ports.UnsareportConfig) error {
	backupPath := filepath.Join(destDir, backupDir)

	if err := s.FS.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		slog.Warn("could not remove old backup", "error", err)
	}

	if err := s.FS.EnsureDir(backupPath); err != nil {
		return fmt.Errorf("ensure backup dir: %w", err)
	}

	var manifest BackupManifest
	manifest.Timestamp = time.Now().Format(time.RFC3339)
	manifest.TemplateVersion = cfg.TemplateVersion

	for _, entry := range entries {
		if entry.Kind != KindFile {
			continue
		}

		srcPath := filepath.Join(destDir, filepath.FromSlash(entry.Dest))
		if !s.FS.FileExists(srcPath) {
			continue
		}

		data, err := s.FS.ReadFile(srcPath)
		if err != nil {
			continue
		}

		backupFile := filepath.Join(backupPath, filepath.FromSlash(entry.Dest))
		if err := s.FS.EnsureDir(filepath.Dir(backupFile)); err != nil {
			continue
		}
		if err := s.FS.WriteFileAtomic(backupFile, data, 0o644); err != nil {
			continue
		}

		manifest.Files = append(manifest.Files, BackupManifestEntry{
			RelativePath: entry.Dest,
			OriginalPath: srcPath,
		})
	}

	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal backup manifest: %w", err)
	}

	manifestPath := filepath.Join(backupPath, "manifest.json")
	if err := s.FS.WriteFileAtomic(manifestPath, manifestData, 0o644); err != nil {
		return fmt.Errorf("write backup manifest: %w", err)
	}

	if _, err := fmt.Fprintf(s.Stdout, "Backup created: %s (%d files)\n", backupDir, len(manifest.Files)); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	return nil
}

func (s *RollbackService) Rollback(destDir string) error {
	backupPath := filepath.Join(destDir, backupDir)
	manifestPath := filepath.Join(backupPath, "manifest.json")

	if !s.FS.FileExists(manifestPath) {
		return fmt.Errorf("no backup found at %s", backupDir)
	}

	data, err := s.FS.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read backup manifest: %w", err)
	}

	var manifest BackupManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse backup manifest: %w", err)
	}

	restored := 0
	var failed []string
	for _, entry := range manifest.Files {
		backupFile := filepath.Join(backupPath, filepath.FromSlash(entry.RelativePath))
		if !s.FS.FileExists(backupFile) {
			failed = append(failed, entry.RelativePath)
			continue
		}

		data, err := s.FS.ReadFile(backupFile)
		if err != nil {
			failed = append(failed, entry.RelativePath)
			continue
		}

		if err := s.FS.WriteFileAtomic(entry.OriginalPath, data, 0o644); err != nil {
			failed = append(failed, entry.RelativePath)
			continue
		}
		restored++
	}

	if err := s.FS.Remove(backupPath); err != nil {
		slog.Warn("could not remove backup after restore", "error", err)
	}

	if _, err := fmt.Fprintf(s.Stdout, "Rollback complete: %d files restored from backup (%s)\n", restored, manifest.Timestamp); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if len(failed) > 0 {
		slog.Warn("some files could not be restored", "count", len(failed), "files", strings.Join(failed, ", "))
	}
	return nil
}

func (s *RollbackService) HasBackup(destDir string) bool {
	manifestPath := filepath.Join(destDir, backupDir, "manifest.json")
	return s.FS.FileExists(manifestPath)
}
