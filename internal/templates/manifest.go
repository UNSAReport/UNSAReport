package templates

import (
	"encoding/json"
	"fmt"
)

type EntryKind string

const (
	KindFile EntryKind = "file"
	KindDir  EntryKind = "dir"
)

type Entry struct {
	Kind       EntryKind `json:"kind"`
	Src        string    `json:"src,omitempty"`
	Dest       string    `json:"dest"`
	AutoUpdate bool      `json:"autoUpdate,omitempty"`
}

type MultiSection struct {
	Readme   Entry   `json:"readme"`
	Root     []Entry `json:"root"`
	LabFiles []Entry `json:"labFiles"`
}

type Manifest struct {
	Common []Entry      `json:"common"`
	Single []Entry      `json:"single"`
	Multi  MultiSection `json:"multi"`
}

func LoadManifest(files Files) (*Manifest, error) {
	data, ok := files["manifest.json"]
	if !ok {
		return nil, fmt.Errorf("template manifest.json not found")
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
