package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd_Help(t *testing.T) {
	ensureCLI()
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "UNSAReport template CLI")
}

func TestRootCmd_NoArgs_ShowsHelp(t *testing.T) {
	ensureCLI()
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "UNSAReport template CLI")
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
			require.NoError(t, err)
			assert.NotEmpty(t, buf.String(), "completion output should not be empty for %s", shell)
		})
	}
}

func TestRootCmd_UnknownFlag(t *testing.T) {
	ensureCLI()
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--nonexistent"})

	err := cmd.Execute()
	require.Error(t, err)
}
