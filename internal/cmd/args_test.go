package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
			assert.Equal(t, tt.wantName, gotName)
			assert.Equal(t, tt.wantRange, gotRange)
		})
	}
}

func TestParseComponentArg(t *testing.T) {
	name, rangeSpec := parseComponentArg("code-block@^1.0.0")
	assert.Equal(t, "code-block", name)
	assert.Equal(t, "^1.0.0", rangeSpec)
}
