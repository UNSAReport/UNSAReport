package services

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// EntryKind represents whether a manifest entry is a file or directory.
type EntryKind string

const (
	// KindFile denotes a file entry in a manifest.
	KindFile EntryKind = "file"
	// KindDir denotes a directory entry in a manifest.
	KindDir EntryKind = "dir"
)

// Entry describes a single file or directory to be installed from a template.
type Entry struct {
	Kind      EntryKind `json:"kind"`
	Src       string    `json:"src,omitempty"`
	Dest      string    `json:"dest"`
	Updatable bool      `json:"updatable,omitempty"`
}

// MultiEntrySet groups entries into root files and per-lab files for multi-mode manifests.
type MultiEntrySet struct {
	Root     []Entry `json:"root"`
	LabFiles []Entry `json:"labFiles"`
}

// Manifest is the top-level parsed representation of a template manifest.
type Manifest struct {
	Mode       string            `json:"mode"`
	Components map[string]string `json:"components,omitempty"`
	Entries    any               `json:"entries"`
}

// SingleManifest represents a manifest in "single" mode with a flat list of entries.
type SingleManifest struct {
	Mode       string            `json:"mode"`
	Components map[string]string `json:"components,omitempty"`
	Entries    []Entry           `json:"entries"`
}

// MultiManifest represents a manifest in "multi" mode with root and lab-file entry sets.
type MultiManifest struct {
	Mode       string            `json:"mode"`
	Components map[string]string `json:"components,omitempty"`
	Entries    MultiEntrySet     `json:"entries"`
}

// Validate checks that the manifest has mode "single" and all entries are well-formed.
func (m *SingleManifest) Validate() error {
	if m.Mode != "single" {
		return fmt.Errorf("manifest mode must be %q, got %q", "single", m.Mode)
	}
	if len(m.Entries) == 0 {
		return fmt.Errorf("manifest entries must not be empty")
	}
	for i, e := range m.Entries {
		if err := validateEntry(e); err != nil {
			return fmt.Errorf("entry[%d]: %w", i, err)
		}
	}
	return nil
}

// Validate checks that the manifest has mode "multi" and both root and lab entries are well-formed.
func (m *MultiManifest) Validate() error {
	if m.Mode != "multi" {
		return fmt.Errorf("manifest mode must be %q, got %q", "multi", m.Mode)
	}
	if len(m.Entries.Root) == 0 {
		return fmt.Errorf("manifest root entries must not be empty")
	}
	if len(m.Entries.LabFiles) == 0 {
		return fmt.Errorf("manifest labFiles entries must not be empty")
	}
	for i, e := range m.Entries.Root {
		if err := validateEntry(e); err != nil {
			return fmt.Errorf("root entry[%d]: %w", i, err)
		}
	}
	for i, e := range m.Entries.LabFiles {
		if err := validateEntry(e); err != nil {
			return fmt.Errorf("labFiles entry[%d]: %w", i, err)
		}
	}
	return nil
}

func validateEntry(e Entry) error {
	switch e.Kind {
	case KindFile:
		if strings.TrimSpace(e.Src) == "" {
			return fmt.Errorf("file entry src must not be empty")
		}
	case KindDir:
		if strings.TrimSpace(e.Dest) == "" {
			return fmt.Errorf("dir entry dest must not be empty")
		}
	default:
		return fmt.Errorf("invalid entry kind %q (must be %q or %q)", e.Kind, KindFile, KindDir)
	}
	return nil
}

// LoadManifest parses raw JSON into a Manifest without validating its entries.
func LoadManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if m.Mode != "single" && m.Mode != "multi" {
		return nil, fmt.Errorf("manifest mode must be %q or %q, got %q", "single", "multi", m.Mode)
	}
	return &m, nil
}

// LoadAndValidateManifest parses raw JSON and validates the manifest entries.
func LoadAndValidateManifest(data []byte) (*Manifest, error) {
	switch {
	case strings.Contains(string(data), `"mode": "single"`):
		var sm SingleManifest
		if err := json.Unmarshal(data, &sm); err != nil {
			return nil, fmt.Errorf("parse single manifest: %w", err)
		}
		if err := sm.Validate(); err != nil {
			return nil, fmt.Errorf("validate manifest: %w", err)
		}
		return &Manifest{Mode: sm.Mode, Components: sm.Components, Entries: sm.Entries}, nil

	case strings.Contains(string(data), `"mode": "multi"`):
		var mm MultiManifest
		if err := json.Unmarshal(data, &mm); err != nil {
			return nil, fmt.Errorf("parse multi manifest: %w", err)
		}
		if err := mm.Validate(); err != nil {
			return nil, fmt.Errorf("validate manifest: %w", err)
		}
		return &Manifest{Mode: mm.Mode, Components: mm.Components, Entries: mm.Entries}, nil

	default:
		return nil, fmt.Errorf("manifest mode must be %q or %q", "single", "multi")
	}
}

// GetComponents returns the component dependency map, or an empty map if none are declared.
func (m *Manifest) GetComponents() map[string]string {
	if m.Components == nil {
		return map[string]string{}
	}
	return m.Components
}

// GetSingleEntries extracts the entries from a single-mode manifest.
func (m *Manifest) GetSingleEntries() ([]Entry, error) {
	if m.Mode != "single" {
		return nil, fmt.Errorf("manifest is not single-mode")
	}
	data, err := json.Marshal(m.Entries)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// GetMultiEntries extracts the entries from a multi-mode manifest.
func (m *Manifest) GetMultiEntries() (*MultiEntrySet, error) {
	if m.Mode != "multi" {
		return nil, fmt.Errorf("manifest is not multi-mode")
	}
	data, err := json.Marshal(m.Entries)
	if err != nil {
		return nil, err
	}
	var entries MultiEntrySet
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return &entries, nil
}

// ExpandDirEntries replaces directory entries with the individual files they contain, based on the remote file map.
func ExpandDirEntries(remote map[string][]byte, entries []Entry) []Entry {
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		out = append(out, e)
		if e.Kind != KindDir || strings.TrimSpace(e.Src) == "" {
			continue
		}

		prefix := strings.TrimSuffix(e.Src, "/") + "/"
		paths := make([]string, 0)
		for p := range remote {
			if strings.HasPrefix(p, prefix) {
				paths = append(paths, p)
			}
		}
		sort.Strings(paths)

		for _, p := range paths {
			rel := strings.TrimPrefix(p, prefix)
			if rel == "" {
				continue
			}
			out = append(out, Entry{
				Kind: KindFile,
				Src:  p,
				Dest: filepath.ToSlash(filepath.Join(e.Dest, rel)),
			})
		}
	}
	return out
}
