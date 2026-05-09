package cmd

import (
	"github.com/christianmz565/lab-report/internal/adapters/config"
	"github.com/christianmz565/lab-report/internal/adapters/osfs"
	"github.com/christianmz565/lab-report/internal/adapters/vhs"
	"github.com/christianmz565/lab-report/internal/services"
	"github.com/spf13/cobra"
)

func newCaptureCmd() *cobra.Command {
	var tapeFile string
	var cwdFlag string

	cmd := &cobra.Command{
		Use:   "capture [flags] <result.png> [instructions...]",
		Short: "Capture terminal output and render it to a PNG via vhs",
		Long: `Capture terminal output and render it to a PNG using charmbracelet/vhs.

This command can be used in oneshot mode by passing the result PNG path and a list
of instructions, or by providing a specific .tape file via the --tape flag.`,
		Example: `  # Oneshot mode
  lab-report capture --cwd my_dir result.png "python" "print('hello')" "tape:Ctrl+D"

  # Run a tape file directly
  lab-report capture --tape template.tape`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if tapeFile == "" && len(args) < 1 {
				return cmd.Help()
			}

			fs := osfs.New()
			renderer := vhs.New()
			cfg := config.New()

			svc := services.NewCaptureService(renderer, fs, cfg)
			return svc.Execute(cmd.Context(), tapeFile, cwdFlag, args)
		},
	}

	cmd.Flags().StringVarP(&tapeFile, "tape", "t", "", "Path to a specific .tape file to run")
	cmd.Flags().StringVar(&cwdFlag, "cwd", "", "Directory to cd into at the start of the capture")

	return cmd
}
