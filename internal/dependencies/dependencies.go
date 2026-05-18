package dependencies

import (
	"fmt"
	"os/exec"
)

type Tool struct {
	Name        string
	Description string
}

var (
	Typst       = Tool{Name: "typst", Description: "Typst compiler"}
	Freeze      = Tool{Name: "freeze", Description: "charmbracelet/freeze (terminal capture renderer)"}
	ImageMagick = Tool{Name: "magick", Description: "ImageMagick (SVG to PNG conversion)"}
)

var Required = []Tool{Typst, Freeze, ImageMagick}

func Check(tool Tool) error {
	if _, err := exec.LookPath(tool.Name); err != nil {
		return fmt.Errorf("missing required external tool on PATH: %s (%s)", tool.Name, tool.Description)
	}
	return nil
}
