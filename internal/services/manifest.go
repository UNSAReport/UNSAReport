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
	Mode    string      `json:"mode"`
	Entries interface{} `json:"entries"`
}

func LoadManifest(files map[string][]byte) (*Manifest, error) {
	data, ok := files["manifest.json"]
	if !ok {
		return nil, fmt.Errorf("template manifest.json not found")
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Mode != "single" && m.Mode != "multi" {
		return nil, fmt.Errorf("invalid manifest mode %q (must be \"single\" or \"multi\")", m.Mode)
	}
	return &m, nil
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
