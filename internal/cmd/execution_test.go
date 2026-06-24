package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd_Execute(t *testing.T) {
	ensureCLI()

	t.Run("no args shows help", func(t *testing.T) {
		cmd := rootCmd
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "UNSAReport template CLI")
	})

	t.Run("--help shows help", func(t *testing.T) {
		cmd := rootCmd
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"--help"})

		err := cmd.Execute()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "UNSAReport template CLI")
	})
}

func TestInstallCmd_Args(t *testing.T) {
	ensureCLI()

	t.Run("too many args shows help", func(t *testing.T) {
		cmd := rootCmd
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"install", "a", "b"})

		err := cmd.Execute()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "install")
	})
}

func TestCaptureCmd_NoArgs(t *testing.T) {
	ensureCLI()

	t.Run("no args shows help", func(t *testing.T) {
		cmd := rootCmd
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"capture"})

		err := cmd.Execute()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "capture")
	})
}

func TestPrepareCmd_TooManyArgs(t *testing.T) {
	ensureCLI()

	t.Run("too many args shows help", func(t *testing.T) {
		cmd := rootCmd
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"prepare", "a", "b"})

		err := cmd.Execute()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "prepare")
	})
}

func TestComponentCmd_Subcommands(t *testing.T) {
	ensureCLI()

	t.Run("no subcommand shows help", func(t *testing.T) {
		cmd := rootCmd
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"component"})

		err := cmd.Execute()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "component")
	})
}

func TestCompletionCmd_AllShells(t *testing.T) {
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

func TestCompletionCmd_InvalidShell(t *testing.T) {
	ensureCLI()

	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"completion", "invalid"})

	err := cmd.Execute()
	if err != nil {
		assert.Contains(t, err.Error(), "invalid argument")
	}
}

func TestRootCmd_SubcommandsRegistered(t *testing.T) {
	ensureCLI()

	expectedCommands := []string{"install", "update", "prepare", "capture", "component", "completion"}
	for _, name := range expectedCommands {
		t.Run(name, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{name})
			require.NoError(t, err, "subcommand %q should exist", name)
			assert.Equal(t, name, cmd.Name())
		})
	}
}
