package services

import (
	"testing"
)

func TestLoadAndValidateManifest_Single(t *testing.T) {
	data := []byte(`{
		"mode": "single",
		"entries": [
			{"kind": "file", "src": "report.typ", "dest": "report.typ"}
		]
	}`)

	m, err := LoadAndValidateManifest(data)
	if err != nil {
		t.Fatalf("LoadAndValidateManifest() error = %v", err)
	}

	if m.Mode != "single" {
		t.Errorf("Mode = %q, want %q", m.Mode, "single")
	}

	entries, err := m.GetSingleEntries()
	if err != nil {
		t.Fatalf("GetSingleEntries() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}

	if entries[0].Dest != "report.typ" {
		t.Errorf("entries[0].Dest = %q, want %q", entries[0].Dest, "report.typ")
	}
}

func TestLoadAndValidateManifest_Multi(t *testing.T) {
	data := []byte(`{
		"mode": "multi",
		"entries": {
			"root": [
				{"kind": "dir", "dest": "common"}
			],
			"labFiles": [
				{"kind": "file", "src": "report.typ", "dest": "{lab}/report.typ"}
			]
		}
	}`)

	m, err := LoadAndValidateManifest(data)
	if err != nil {
		t.Fatalf("LoadAndValidateManifest() error = %v", err)
	}

	if m.Mode != "multi" {
		t.Errorf("Mode = %q, want %q", m.Mode, "multi")
	}

	entries, err := m.GetMultiEntries()
	if err != nil {
		t.Fatalf("GetMultiEntries() error = %v", err)
	}

	if len(entries.Root) != 1 {
		t.Fatalf("len(root) = %d, want 1", len(entries.Root))
	}

	if len(entries.LabFiles) != 1 {
		t.Fatalf("len(labFiles) = %d, want 1", len(entries.LabFiles))
	}
}

func TestLoadAndValidateManifest_InvalidMode(t *testing.T) {
	data := []byte(`{"mode": "invalid"}`)
	_, err := LoadAndValidateManifest(data)
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestLoadAndValidateManifest_EmptyEntries(t *testing.T) {
	data := []byte(`{"mode": "single", "entries": []}`)
	_, err := LoadAndValidateManifest(data)
	if err == nil {
		t.Fatal("expected error for empty entries")
	}
}

func TestExpandDirEntries(t *testing.T) {
	remote := map[string][]byte{
		"common/style.typ":     []byte("style"),
		"common/utils.typ":     []byte("utils"),
		"common/deep/file.typ": []byte("file"),
	}

	entries := []Entry{
		{Kind: KindDir, Src: "common", Dest: "output/common"},
	}

	expanded := ExpandDirEntries(remote, entries)

	// 1 dir entry + 3 expanded file entries = 4 total
	if len(expanded) != 4 {
		t.Fatalf("len(expanded) = %d, want 4", len(expanded))
	}

	// First entry is the original dir
	if expanded[0].Kind != KindDir {
		t.Errorf("expanded[0].Kind = %q, want %q", expanded[0].Kind, KindDir)
	}
}

func TestSubstituteLab(t *testing.T) {
	entries := []Entry{
		{Kind: KindFile, Src: "report.typ", Dest: "{lab}/report.typ"},
		{Kind: KindFile, Src: "main.typ", Dest: "{lab}/main.typ"},
	}

	result := substituteLab(entries, "l1")

	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}

	if result[0].Dest != "l1/report.typ" {
		t.Errorf("result[0].Dest = %q, want %q", result[0].Dest, "l1/report.typ")
	}

	if result[1].Dest != "l1/main.typ" {
		t.Errorf("result[1].Dest = %q, want %q", result[1].Dest, "l1/main.typ")
	}
}

func TestGetComponents(t *testing.T) {
	t.Run("nil components", func(t *testing.T) {
		m := &Manifest{Mode: "single"}
		components := m.GetComponents()
		if len(components) != 0 {
			t.Errorf("expected empty map, got %v", components)
		}
	})

	t.Run("with components", func(t *testing.T) {
		m := &Manifest{
			Mode:       "single",
			Components: map[string]string{"code-block": "^1.0.0"},
		}
		components := m.GetComponents()
		if components["code-block"] != "^1.0.0" {
			t.Errorf("expected ^1.0.0, got %s", components["code-block"])
		}
	})
}
