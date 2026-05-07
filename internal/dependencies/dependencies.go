package dependencies

import (
	"fmt"
	"os/exec"
	"strings"
)

type Tool struct {
	Name        string
	Description string
}

var Required = []Tool{
	{Name: "typst", Description: "Typst compiler"},
	{Name: "freeze", Description: "charmbracelet/freeze (terminal capture renderer)"},
	{Name: "magick", Description: "ImageMagick (SVG -> PNG conversion)"},
}

func CheckAll() error {
	missing := make([]Tool, 0)
	for _, tool := range Required {
		if _, err := exec.LookPath(tool.Name); err != nil {
			missing = append(missing, tool)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString("missing required external tools on PATH:\n")
	for _, m := range missing {
		b.WriteString(fmt.Sprintf("  - %s (%s)\n", m.Name, m.Description))
	}
	b.WriteString("\nHint: install them (or use the provided Nix flake dev shell).\n")
	return fmt.Errorf("%s", b.String())
}
