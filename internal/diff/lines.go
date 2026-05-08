package diff

import (
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// UnifiedLineDiff returns a simple, line-oriented diff with prefixes:
//
//	' ' unchanged
//	'-' removed
//	'+' added
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
