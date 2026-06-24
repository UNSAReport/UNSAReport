package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean filename",
			input:    "hello-world",
			expected: "hello-world",
		},
		{
			name:     "filename with illegal chars",
			input:    "hello:world/test",
			expected: "hello-world-test",
		},
		{
			name:     "filename with angle brackets",
			input:    "<test>",
			expected: "-test-",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "filename with quotes",
			input:    `test"file`,
			expected: "test-file",
		},
		{
			name:     "filename with pipe",
			input:    "file|name",
			expected: "file-name",
		},
		{
			name:     "filename with question mark",
			input:    "file?name",
			expected: "file-name",
		},
		{
			name:     "filename with asterisk",
			input:    "file*name",
			expected: "file-name",
		},
		{
			name:     "filename with backslash",
			input:    `file\name`,
			expected: "file-name",
		},
		{
			name:     "multiple illegal chars",
			input:    `<>"|?*`,
			expected: "------",
		},
		{
			name:     "spaces preserved",
			input:    "hello world",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestApplyTemplate(t *testing.T) {
	t.Parallel()

	vars := map[string]string{
		"lab_number": "1",
		"course":     "SO",
		"members":    "John Doe;Jane Smith",
	}

	tests := []struct {
		name       string
		tpl        string
		outputType string
		expected   string
	}{
		{
			name:       "simple template",
			tpl:        "{output_type}_{lab_number}",
			outputType: "Informe",
			expected:   "Informe_1",
		},
		{
			name:       "template with course",
			tpl:        "{output_type}_{course}_{lab_number}",
			outputType: "Informe",
			expected:   "Informe_SO_1",
		},
		{
			name:       "template with unknown var",
			tpl:        "{output_type}_{unknown}",
			outputType: "Informe",
			expected:   "Informe_{unknown}",
		},
		{
			name:       "empty template",
			tpl:        "",
			outputType: "Informe",
			expected:   "",
		},
		{
			name:       "template with no vars",
			tpl:        "static_name",
			outputType: "Informe",
			expected:   "static_name",
		},
		{
			name:       "output_type with illegal chars",
			tpl:        "{output_type}",
			outputType: "Código Fuente",
			expected:   "Código Fuente",
		},
		{
			name:       "multiple same var",
			tpl:        "{lab_number}_{lab_number}",
			outputType: "Informe",
			expected:   "1_1",
		},
		{
			name:       "empty output type",
			tpl:        "{output_type}_{lab_number}",
			outputType: "",
			expected:   "_1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ApplyTemplate(tt.tpl, vars, tt.outputType)
			assert.Equal(t, tt.expected, got)
		})
	}
}
