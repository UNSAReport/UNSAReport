package registry

import (
	"testing"

	"github.com/Masterminds/semver/v3"
)

func TestResolveVersionFromMap(t *testing.T) {
	v1, _ := semver.NewVersion("1.0.0")
	v1_1, _ := semver.NewVersion("1.1.0")
	v2, _ := semver.NewVersion("2.0.0")

	availableVersions := map[string]*semver.Version{
		"1.0.0": v1,
		"1.1.0": v1_1,
		"2.0.0": v2,
	}

	distTags := map[string]*semver.Version{
		"latest": v2,
	}

	tests := []struct {
		name      string
		rangeSpec string
		expected  string
		wantErr   bool
	}{
		{
			name:      "resolve latest",
			rangeSpec: "latest",
			expected:  "2.0.0",
		},
		{
			name:      "resolve empty string as latest",
			rangeSpec: "",
			expected:  "2.0.0",
		},
		{
			name:      "resolve wildcard",
			rangeSpec: "*",
			expected:  "2.0.0",
		},
		{
			name:      "resolve caret range",
			rangeSpec: "^1.0.0",
			expected:  "1.1.0",
		},
		{
			name:      "resolve exact version",
			rangeSpec: "1.0.0",
			expected:  "1.0.0",
		},
		{
			name:      "no matching version",
			rangeSpec: ">=3.0.0",
			wantErr:   true,
		},
		{
			name:      "invalid range",
			rangeSpec: "not-a-version",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveVersionFromMap(availableVersions, distTags, tt.rangeSpec)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveVersionFromMap() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got.String() != tt.expected {
				t.Errorf("resolveVersionFromMap() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestResolveLatestFromMap(t *testing.T) {
	v1, _ := semver.NewVersion("1.0.0")
	v2, _ := semver.NewVersion("2.0.0")
	pre, _ := semver.NewVersion("3.0.0-beta.1")

	versions := map[string]*semver.Version{
		"1.0.0":        v1,
		"2.0.0":        v2,
		"3.0.0-beta.1": pre,
	}

	got, err := resolveLatestFromMap(versions)
	if err != nil {
		t.Fatalf("resolveLatestFromMap() error = %v", err)
	}

	if got.String() != "2.0.0" {
		t.Errorf("resolveLatestFromMap() = %v, want 2.0.0", got)
	}
}

func TestResolveLatestFromMap_NoStable(t *testing.T) {
	pre, _ := semver.NewVersion("1.0.0-beta.1")
	versions := map[string]*semver.Version{
		"1.0.0-beta.1": pre,
	}

	_, err := resolveLatestFromMap(versions)
	if err == nil {
		t.Fatal("expected error for no stable versions")
	}
}
