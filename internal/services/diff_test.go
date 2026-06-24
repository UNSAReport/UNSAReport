package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnifiedLineDiff(t *testing.T) {
	t.Parallel()

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
		{
			name:     "empty strings",
			old:      "",
			new:      "",
			expected: "",
		},
		{
			name:     "old empty",
			old:      "",
			new:      "new line\n",
			expected: "+new line\n",
		},
		{
			name:     "new empty",
			old:      "old line\n",
			new:      "",
			expected: "-old line\n",
		},
		{
			name: "multiple changes",
			old:  "a\nb\nc\n",
			new:  "a\nx\nc\n",
			expected: " a\n" +
				"-b\n" +
				"+x\n" +
				" c\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := UnifiedLineDiff(tt.old, tt.new)
			assert.Equal(t, tt.expected, got)
		})
	}
}
