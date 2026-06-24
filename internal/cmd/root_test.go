package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCmd_Help(t *testing.T) {
	ensureCLI()
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "UNSAReport template CLI") {
		t.Errorf("expected help output to contain description, got: %s", output)
	}
}

func TestRootCmd_NoArgs_ShowsHelp(t *testing.T) {
	ensureCLI()
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "UNSAReport template CLI") {
		t.Errorf("expected help output, got: %s", output)
	}
}

func TestCompletionCmd(t *testing.T) {
	ensureCLI()
	shells := []string{"bash", "zsh", "fish", "powershell"}

	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			cmd := rootCmd
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs([]string{"completion", shell})

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			output := buf.String()
			if len(output) == 0 {
				t.Errorf("expected non-empty completion output for %s", shell)
			}
		})
	}
}
