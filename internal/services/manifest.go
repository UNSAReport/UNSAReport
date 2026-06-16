package services

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type EntryKind string

const (
	KindFile EntryKind = "file"
	KindDir  EntryKind = "dir"
)

type Entry struct {
	Kind      EntryKind `json:"kind"`
	Src       string    `json:"src,omitempty"`
	Dest      string    `json:"dest"`
	Updatable bool      `json:"updatable,omitempty"`
}

type MultiEntrySet struct {
	Root     []Entry `json:"root"`
	LabFiles []Entry `json:"labFiles"`
}

type Manifest struct {
	Version string      `json:"version,omitempty"`
	Mode    string      `json:"mode"`
	Entries interface{} `json:"entries"`
}

type SingleManifest struct {
	Version string  `json:"version,omitempty"`
	Mode    string  `json:"mode"`
	Entries []Entry `json:"entries"`
}

type MultiManifest struct {
	Version string         `json:"version,omitempty"`
	Mode    string         `json:"mode"`
	Entries MultiEntrySet  `json:"entries"`
}

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
		return &Manifest{Version: sm.Version, Mode: sm.Mode, Entries: sm.Entries}, nil

	case strings.Contains(string(data), `"mode": "multi"`):
		var mm MultiManifest
		if err := json.Unmarshal(data, &mm); err != nil {
			return nil, fmt.Errorf("parse multi manifest: %w", err)
		}
		if err := mm.Validate(); err != nil {
			return nil, fmt.Errorf("validate manifest: %w", err)
		}
		return &Manifest{Version: mm.Version, Mode: mm.Mode, Entries: mm.Entries}, nil

	default:
		return nil, fmt.Errorf("manifest mode must be %q or %q", "single", "multi")
	}
}

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
