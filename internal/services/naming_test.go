package services

import (
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestApplyTemplate(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyTemplate(tt.tpl, vars, tt.outputType)
			if got != tt.expected {
				t.Errorf("ApplyTemplate(%q, vars, %q) = %q, want %q", tt.tpl, tt.outputType, got, tt.expected)
			}
		})
	}
}
