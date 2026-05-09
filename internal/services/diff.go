package services

import (
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func UnifiedLineDiff(oldText, newText string) string {
	dmp := diffmatchpatch.New()
	a, b, lines := dmp.DiffLinesToChars(oldText, newText)
	diffs := dmp.DiffMain(a, b, false)
	diffs = dmp.DiffCharsToLines(diffs, lines)
	dmp.DiffCleanupSemantic(diffs)

	var out strings.Builder
	for _, d := range diffs {
		prefix := " "
		switch d.Type {
		case diffmatchpatch.DiffDelete:
			prefix = "-"
		case diffmatchpatch.DiffInsert:
			prefix = "+"
		}

		for line := range strings.SplitSeq(d.Text, "\n") {
			line = strings.TrimRight(line, "\r")
			if line == "" {
				continue
			}
			out.WriteString(prefix)
			out.WriteString(line)
			out.WriteByte('\n')
		}
	}
	return out.String()
}
