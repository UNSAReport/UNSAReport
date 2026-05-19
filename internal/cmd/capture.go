package cmd

import (
	"fmt"

	"github.com/christianmz565/lab-report/internal/adapters/config"
	"github.com/christianmz565/lab-report/internal/adapters/freeze"
	"github.com/christianmz565/lab-report/internal/adapters/osfs"
	"github.com/christianmz565/lab-report/internal/services"
	"github.com/mattn/go-shellwords"
	"github.com/spf13/cobra"
)

func newCaptureCmd() *cobra.Command {
	var cwdFlag string
	var freezeFlags string
	var saveFlags bool

	cmd := &cobra.Command{
		Use:   "capture [flags] <result.png> [instructions...]",
		Short: "Capture terminal output and render it to a PNG via freeze",
		Long: `Capture terminal output and render it to a PNG using charmbracelet/freeze and ImageMagick.

This command executes terminal instructions directly in a virtual terminal and captures the
resulting output. It applies custom prompt formatting defined in the labreport.json configuration.

Instructions are processed sequentially:
- Regular text: Typed into the terminal followed by Enter.
- "w:<duration>": Delays execution (e.g., "w:2s", "w:500ms").
- "r:<text>": Writes text directly without Enter.
- "c:<key>": Sends Ctrl + key combination (e.g., "c:c" for Ctrl+C).
- "k:<key>": Sends a control key (e.g., "k:enter", "k:tab", "k:backspace", "k:esc").

A raw log of the session (including ANSI colors) is automatically saved in
the capture_logs/ directory as a .log file.`,
		Example: `  # Simple capture
  lab-report capture output.png "ls -la" "cat README.md"

  # Using control characters and raw input
  lab-report capture output.png "python" "print('hello')" "k:enter" "c:d"

  # With custom directory and delays
  lab-report capture --cwd ./src result.png "python" "print('hello')" "w:1s"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return cmd.Help()
			}

			fs := osfs.New()
			renderer := freeze.New()
			cfg := config.New()

			var parsedFlags []string
			if freezeFlags != "" {
				var err error
				parsedFlags, err = shellwords.Parse(freezeFlags)
				if err != nil {
					return fmt.Errorf("parse freeze-flags: %w", err)
				}
			}

			svc := services.NewCaptureService(renderer, fs, cfg)
			return svc.Execute(cmd.Context(), services.CaptureOptions{
				Cwd:             cwdFlag,
				Args:            args,
				FreezeFlags:     parsedFlags,
				SaveFreezeFlags: saveFlags,
			})
		},
	}

	cmd.Flags().StringVar(&cwdFlag, "cwd", "", "Directory to cd into at the start of the capture")
	cmd.Flags().StringVar(&freezeFlags, "freeze-flags", "", "Additional flags to pass to freeze (e.g., \"--theme dracula\")")
	cmd.Flags().BoolVar(&saveFlags, "save-freeze-flags", false, "Save the passed freeze-flags to labreport.json")

	return cmd
}
