package services

import (
	"regexp"
)

var reVar = regexp.MustCompile(`\{(\w+)\}`)
var reIllegal = regexp.MustCompile(`[<>:"/\\|?*]`)

// SanitizeFilename replaces characters that are illegal in file paths with hyphens.
func SanitizeFilename(s string) string {
	return reIllegal.ReplaceAllString(s, "-")
}

// ApplyTemplate expands {variable} placeholders in tpl using vars, treating {output_type} as outputType.
func ApplyTemplate(tpl string, vars map[string]string, outputType string) string {
	return reVar.ReplaceAllStringFunc(tpl, func(m string) string {
		sub := reVar.FindStringSubmatch(m)
		if len(sub) != 2 {
			return m
		}
		key := sub[1]
		if key == "output_type" {
			return SanitizeFilename(outputType)
		}
		if v, ok := vars[key]; ok {
			return SanitizeFilename(v)
		}
		return m
	})
}
