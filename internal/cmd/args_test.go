package cmd

import (
	"testing"
)

func TestParseArg(t *testing.T) {
	tests := []struct {
		name      string
		arg       string
		wantName  string
		wantRange string
	}{
		{
			name:      "name only",
			arg:       "lab",
			wantName:  "lab",
			wantRange: "latest",
		},
		{
			name:      "name with version",
			arg:       "lab@1.0.0",
			wantName:  "lab",
			wantRange: "1.0.0",
		},
		{
			name:      "name with range",
			arg:       "lab@^1.0.0",
			wantName:  "lab",
			wantRange: "^1.0.0",
		},
		{
			name:      "empty string",
			arg:       "",
			wantName:  "",
			wantRange: "latest",
		},
		{
			name:      "at sign only",
			arg:       "@latest",
			wantName:  "",
			wantRange: "latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotRange := parseArg(tt.arg)
			if gotName != tt.wantName {
				t.Errorf("parseArg(%q) name = %q, want %q", tt.arg, gotName, tt.wantName)
			}
			if gotRange != tt.wantRange {
				t.Errorf("parseArg(%q) range = %q, want %q", tt.arg, gotRange, tt.wantRange)
			}
		})
	}
}

func TestParseComponentArg(t *testing.T) {
	name, rangeSpec := parseComponentArg("code-block@^1.0.0")
	if name != "code-block" {
		t.Errorf("name = %q, want code-block", name)
	}
	if rangeSpec != "^1.0.0" {
		t.Errorf("rangeSpec = %q, want ^1.0.0", rangeSpec)
	}
}
