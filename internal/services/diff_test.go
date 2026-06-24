package services

import (
	"testing"
)

func TestUnifiedLineDiff(t *testing.T) {
	tests := []struct {
		name     string
		old      string
		new      string
		expected string
	}{
		{
			name:     "identical text",
			old:      "hello\nworld\n",
			new:      "hello\nworld\n",
			expected: " hello\n world\n",
		},
		{
			name: "added line",
			old:  "hello\n",
			new:  "hello\nworld\n",
			expected: " hello\n" +
				"+world\n",
		},
		{
			name: "removed line",
			old:  "hello\nworld\n",
			new:  "hello\n",
			expected: " hello\n" +
				"-world\n",
		},
		{
			name: "changed line",
			old:  "hello\n",
			new:  "goodbye\n",
			expected: "-hello\n" +
				"+goodbye\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UnifiedLineDiff(tt.old, tt.new)
			if got != tt.expected {
				t.Errorf("UnifiedLineDiff() = %q, want %q", got, tt.expected)
			}
		})
	}
}
