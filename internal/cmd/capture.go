package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/christianmz565/lab-report/internal/config"
	"github.com/christianmz565/lab-report/internal/fsutil"
	"github.com/spf13/cobra"
)

type captureOptions struct{}

func newCaptureCmd() *cobra.Command {
	opt := captureOptions{}

	cmd := &cobra.Command{
		Use:   "capture <tape-file>",
		Short: "Capture terminal output and render it to a PNG via vhs",
		Long: `Capture terminal output and render it to a PNG using charmbracelet/vhs.

This command takes a .tape script file as input, executes it in a virtual terminal,
and saves the result (usually as a PNG screenshot).`,
		Example: `  # Capture output using a tape file
  lab-report capture template.tape`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return cmd.Help()
			}
			return runCapture(cmd.Context(), opt, args)
		},
	}

	return cmd
}

func runCapture(ctx context.Context, opt captureOptions, args []string) error {
	tapeFile := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	projectRoot, _, ok, err := config.FindProjectRoot(cwd)
	if !ok {
		projectRoot = cwd
	}

	if err := os.Chdir(projectRoot); err != nil {
		return err
	}

	configTapePath := "config.tape"
	if !fsutil.FileExists(configTapePath) {
		configTapeContent := "Set Width 1000\nSet TypingSpeed 0.1\n"
		if err := os.WriteFile(configTapePath, []byte(configTapeContent), 0644); err != nil {
			return err
		}

		templateTapePath := "template.tape"
		if !fsutil.FileExists(templateTapePath) {
			templateContent := "Source config.tape\n\nType \"echo 'Hello from VHS!'\"\nEnter\nSleep 1s\n\nScreenshot output.png\n"
			if err := os.WriteFile(templateTapePath, []byte(templateContent), 0644); err != nil {
				return err
			}
		}

		fmt.Println("config.tape was not found.")
		fmt.Println("Created config.tape and template.tape with defaults in the project root.")
		fmt.Println("Please review them and run the command again.")
		os.Exit(0)
	}

	tapePathAbs, err := filepath.Abs(filepath.Join(cwd, tapeFile))
	if err != nil {
		return err
	}

	vhsCmd := exec.CommandContext(ctx, "vhs", tapePathAbs)
	vhsCmd.Stdin = os.Stdin
	vhsCmd.Stdout = os.Stdout
	vhsCmd.Stderr = os.Stderr

	if err := vhsCmd.Run(); err != nil {
		return fmt.Errorf("vhs command failed: %w", err)
	}

	return nil
}
