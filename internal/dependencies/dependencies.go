package dependencies

import (
	"fmt"
	"os/exec"
)

// Tool represents an external CLI tool dependency required for report generation.
type Tool struct {
	Name        string
	Description string
}

// Well-known external tools used by UNSAReport commands.
var (
	// Typst is the Typst compiler used to render .typ templates into PDF.
	Typst = Tool{Name: "typst", Description: "Typst compiler"}
	// Freeze is charmbracelet/freeze, used to capture styled terminal output as images.
	Freeze = Tool{Name: "freeze", Description: "charmbracelet/freeze (terminal capture renderer)"}
	// ImageMagick is used to convert SVG images into PNG for embedding in reports.
	ImageMagick = Tool{Name: "magick", Description: "ImageMagick (SVG to PNG conversion)"}
)

// Required lists all external tools that must be available on PATH for report generation.
var Required = []Tool{Typst, Freeze, ImageMagick}

// Check verifies that tool is available on the system PATH, returning an error if not found.
func Check(tool Tool) error {
	if _, err := exec.LookPath(tool.Name); err != nil {
		return fmt.Errorf("missing required external tool on PATH: %s (%s)", tool.Name, tool.Description)
	}
	return nil
}
