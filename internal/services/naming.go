package services

import (
	"regexp"
)

var reVar = regexp.MustCompile(`\{(\w+)\}`)

func ApplyTemplate(tpl string, vars map[string]string, outputType string) string {
	return reVar.ReplaceAllStringFunc(tpl, func(m string) string {
		sub := reVar.FindStringSubmatch(m)
		if len(sub) == 2 {
			key := sub[1]
			if key == "outputType" {
				return outputType
			}
			if v, ok := vars[key]; ok {
				return v
			}
		}
		return m
	})
}
