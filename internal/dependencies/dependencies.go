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
	Typst = Tool{Name: "typst", Description: "Typst compiler"}
	VHS   = Tool{Name: "vhs", Description: "charmbracelet/vhs (terminal capture renderer)"}
)

var Required = []Tool{Typst, VHS}

func Check(tool Tool) error {
	if _, err := exec.LookPath(tool.Name); err != nil {
		return fmt.Errorf("missing required external tool on PATH: %s (%s)", tool.Name, tool.Description)
	}
	return nil
}
