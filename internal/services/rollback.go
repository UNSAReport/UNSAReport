package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
}

func NewRollbackService(fs ports.FileSystem, cfg ports.ConfigStore) *RollbackService {
	return &RollbackService{FS: fs, Config: cfg}
}

func (s *RollbackService) CreateBackup(destDir string, entries []Entry, cfg ports.UnsareportConfig) error {
	backupPath := filepath.Join(destDir, backupDir)

	s.FS.Remove(backupPath)

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

	fmt.Fprintf(os.Stdout, "Backup created: %s (%d files)\n", backupDir, len(manifest.Files))
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
	for _, entry := range manifest.Files {
		backupFile := filepath.Join(backupPath, filepath.FromSlash(entry.RelativePath))
		if !s.FS.FileExists(backupFile) {
			continue
		}

		data, err := s.FS.ReadFile(backupFile)
		if err != nil {
			continue
		}

		if err := s.FS.WriteFileAtomic(entry.OriginalPath, data, 0o644); err != nil {
			continue
		}
		restored++
	}

	s.FS.Remove(backupPath)

	fmt.Fprintf(os.Stdout, "Rollback complete: %d files restored from backup (%s)\n", restored, manifest.Timestamp)
	return nil
}

func (s *RollbackService) HasBackup(destDir string) bool {
	manifestPath := filepath.Join(destDir, backupDir, "manifest.json")
	return s.FS.FileExists(manifestPath)
}
