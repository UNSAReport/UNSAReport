package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadAndValidateManifest_Single(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"mode": "single",
		"entries": [
			{"kind": "file", "src": "report.typ", "dest": "report.typ"}
		]
	}`)

	m, err := LoadAndValidateManifest(data)
	require.NoError(t, err)
	assert.Equal(t, "single", m.Mode)

	entries, err := m.GetSingleEntries()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "report.typ", entries[0].Dest)
}

func TestLoadAndValidateManifest_Multi(t *testing.T) {
	t.Parallel()

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
	require.NoError(t, err)
	assert.Equal(t, "multi", m.Mode)

	entries, err := m.GetMultiEntries()
	require.NoError(t, err)
	require.Len(t, entries.Root, 1)
	require.Len(t, entries.LabFiles, 1)
}

func TestLoadAndValidateManifest_InvalidMode(t *testing.T) {
	t.Parallel()

	data := []byte(`{"mode": "invalid"}`)
	_, err := LoadAndValidateManifest(data)
	require.Error(t, err)
}

func TestLoadAndValidateManifest_EmptyEntries(t *testing.T) {
	t.Parallel()

	data := []byte(`{"mode": "single", "entries": []}`)
	_, err := LoadAndValidateManifest(data)
	require.Error(t, err)
}

func TestExpandDirEntries_Manifest(t *testing.T) {
	t.Parallel()

	remote := map[string][]byte{
		"common/style.typ":     []byte("style"),
		"common/utils.typ":     []byte("utils"),
		"common/deep/file.typ": []byte("file"),
	}

	entries := []Entry{
		{Kind: KindDir, Src: "common", Dest: "output/common"},
	}

	expanded := ExpandDirEntries(remote, entries)

	require.Len(t, expanded, 4)
	assert.Equal(t, KindDir, expanded[0].Kind)
}

func TestSubstituteLab_Manifest(t *testing.T) {
	t.Parallel()

	entries := []Entry{
		{Kind: KindFile, Src: "report.typ", Dest: "{lab}/report.typ"},
		{Kind: KindFile, Src: "main.typ", Dest: "{lab}/main.typ"},
	}

	result := substituteLab(entries, "l1")

	require.Len(t, result, 2)
	assert.Equal(t, "l1/report.typ", result[0].Dest)
	assert.Equal(t, "l1/main.typ", result[1].Dest)
}

func TestGetComponents(t *testing.T) {
	t.Parallel()

	t.Run("nil components", func(t *testing.T) {
		t.Parallel()
		m := &Manifest{Mode: "single"}
		components := m.GetComponents()
		assert.Empty(t, components)
	})

	t.Run("with components", func(t *testing.T) {
		t.Parallel()
		m := &Manifest{
			Mode:       "single",
			Components: map[string]string{"code-block": "^1.0.0"},
		}
		components := m.GetComponents()
		assert.Equal(t, "^1.0.0", components["code-block"])
	})

	t.Run("nil components map", func(t *testing.T) {
		t.Parallel()
		m := &Manifest{Mode: "single", Components: nil}
		c := m.GetComponents()
		assert.Empty(t, c)
	})
}

func TestLoadManifest(t *testing.T) {
	t.Parallel()

	t.Run("valid single", func(t *testing.T) {
		t.Parallel()
		data := []byte(`{"mode": "single", "entries": [{"kind": "file", "src": "a", "dest": "b"}]}`)
		m, err := LoadManifest(data)
		require.NoError(t, err)
		assert.Equal(t, "single", m.Mode)
	})

	t.Run("valid multi", func(t *testing.T) {
		t.Parallel()
		data := []byte(`{"mode": "multi", "entries": {"root": [], "labFiles": []}}`)
		m, err := LoadManifest(data)
		require.NoError(t, err)
		assert.Equal(t, "multi", m.Mode)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()
		data := []byte(`{invalid json}`)
		_, err := LoadManifest(data)
		require.Error(t, err)
	})

	t.Run("unknown mode", func(t *testing.T) {
		t.Parallel()
		data := []byte(`{"mode": "triple"}`)
		_, err := LoadManifest(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be")
	})
}

func TestSingleManifest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("wrong mode", func(t *testing.T) {
		t.Parallel()
		m := &SingleManifest{Mode: "multi", Entries: []Entry{{Kind: KindFile, Src: "a", Dest: "b"}}}
		err := m.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be")
	})

	t.Run("empty entries", func(t *testing.T) {
		t.Parallel()
		m := &SingleManifest{Mode: "single", Entries: []Entry{}}
		err := m.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not be empty")
	})

	t.Run("file entry without src", func(t *testing.T) {
		t.Parallel()
		m := &SingleManifest{Mode: "single", Entries: []Entry{{Kind: KindFile, Dest: "b"}}}
		err := m.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "src must not be empty")
	})

	t.Run("dir entry without dest", func(t *testing.T) {
		t.Parallel()
		m := &SingleManifest{Mode: "single", Entries: []Entry{{Kind: KindDir, Src: "a"}}}
		err := m.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dest must not be empty")
	})

	t.Run("invalid entry kind", func(t *testing.T) {
		t.Parallel()
		m := &SingleManifest{Mode: "single", Entries: []Entry{{Kind: "invalid", Src: "a", Dest: "b"}}}
		err := m.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid entry kind")
	})
}

func TestMultiManifest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("empty root", func(t *testing.T) {
		t.Parallel()
		m := &MultiManifest{Mode: "multi", Entries: MultiEntrySet{
			Root:     []Entry{},
			LabFiles: []Entry{{Kind: KindFile, Src: "a", Dest: "b"}},
		}}
		err := m.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "root entries must not be empty")
	})

	t.Run("empty labFiles", func(t *testing.T) {
		t.Parallel()
		m := &MultiManifest{Mode: "multi", Entries: MultiEntrySet{
			Root:     []Entry{{Kind: KindDir, Dest: "common"}},
			LabFiles: []Entry{},
		}}
		err := m.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "labFiles entries must not be empty")
	})

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		m := &MultiManifest{Mode: "multi", Entries: MultiEntrySet{
			Root:     []Entry{{Kind: KindDir, Dest: "common"}},
			LabFiles: []Entry{{Kind: KindFile, Src: "a", Dest: "{lab}/a"}},
		}}
		err := m.Validate()
		require.NoError(t, err)
	})
}

func TestManifest_GetSingleEntries_WrongMode(t *testing.T) {
	t.Parallel()
	m := &Manifest{Mode: "multi"}
	_, err := m.GetSingleEntries()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not single-mode")
}

func TestManifest_GetMultiEntries_WrongMode(t *testing.T) {
	t.Parallel()
	m := &Manifest{Mode: "single"}
	_, err := m.GetMultiEntries()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not multi-mode")
}

func TestExpandDirEntries_NestedDirs(t *testing.T) {
	t.Parallel()

	remote := map[string][]byte{
		"images/photos/cat.jpg": []byte("cat"),
		"images/photos/dog.jpg": []byte("dog"),
		"images/icons/logo.png": []byte("logo"),
		"images/README.md":      []byte("readme"),
	}

	entries := []Entry{
		{Kind: KindDir, Src: "images", Dest: "output/images"},
	}

	expanded := ExpandDirEntries(remote, entries)

	require.Equal(t, 5, len(expanded))
	assert.Equal(t, KindDir, expanded[0].Kind)

	dests := make(map[string]bool)
	for _, e := range expanded[1:] {
		dests[e.Dest] = true
	}
	assert.True(t, dests["output/images/README.md"])
	assert.True(t, dests["output/images/photos/cat.jpg"])
	assert.True(t, dests["output/images/photos/dog.jpg"])
	assert.True(t, dests["output/images/icons/logo.png"])
}

func TestExpandDirEntries_EmptyRemote(t *testing.T) {
	t.Parallel()

	remote := map[string][]byte{}
	entries := []Entry{
		{Kind: KindDir, Src: "common", Dest: "output/common"},
	}

	expanded := ExpandDirEntries(remote, entries)
	assert.Equal(t, 1, len(expanded))
}

func TestLoadAndValidateManifest_MissingEntries(t *testing.T) {
	t.Parallel()
	data := []byte(`{"mode": "single"}`)
	_, err := LoadAndValidateManifest(data)
	require.Error(t, err)
}
