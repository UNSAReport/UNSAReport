package services

import (
	"regexp"
)

var reVar = regexp.MustCompile(`\{(\w+)\}`)
var reIllegal = regexp.MustCompile(`[<>:"/\\|?*]`)

func SanitizeFilename(s string) string {
	return reIllegal.ReplaceAllString(s, "-")
}

func ApplyTemplate(tpl string, vars map[string]string, outputType string) string {
	return reVar.ReplaceAllStringFunc(tpl, func(m string) string {
		sub := reVar.FindStringSubmatch(m)
		if len(sub) == 2 {
			key := sub[1]
			var val string
			if key == "output_type" {
				val = outputType
			} else if v, ok := vars[key]; ok {
				val = v
			} else {
				return m
			}
			return SanitizeFilename(val)
		}
		return m
	})
}
