package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/UNSAReport/UNSAReport/internal/ports"
)

// ReadLockfile reads unsareport.lock from destDir, returning an empty lockfile when the file does not exist.
func (a *Adapter) ReadLockfile(destDir string) (ports.Lockfile, error) {
	path := filepath.Join(destDir, "unsareport.lock")
	var lf ports.Lockfile

	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ports.Lockfile{
				LockfileVersion: "1",
				Packages:        make(map[string]ports.LockfilePackage),
			}, nil
		}
		return ports.Lockfile{}, fmt.Errorf("read lockfile: %w", err)
	}

	if err := json.Unmarshal(b, &lf); err != nil {
		return ports.Lockfile{}, fmt.Errorf("parse lockfile: %w", err)
	}

	if lf.Packages == nil {
		lf.Packages = make(map[string]ports.LockfilePackage)
	}

	return lf, nil
}

// WriteLockfile marshals lf to JSON and writes it to unsareport.lock in destDir.
func (a *Adapter) WriteLockfile(destDir string, lf ports.Lockfile) error {
	path := filepath.Join(destDir, "unsareport.lock")

	if lf.Packages == nil {
		lf.Packages = make(map[string]ports.LockfilePackage)
	}

	b, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal lockfile: %w", err)
	}
	b = append(b, '\n')

	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write lockfile: %w", err)
	}
	return nil
}

// ComputeIntegrity returns a base64-encoded SHA-256 hash of data, prefixed with "sha256-".
func ComputeIntegrity(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256-" + hex.EncodeToString(h[:])
}
