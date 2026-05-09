package cmd

import (
	"github.com/christianmz565/lab-report/internal/adapters/config"
	"github.com/christianmz565/lab-report/internal/adapters/osfs"
	"github.com/christianmz565/lab-report/internal/adapters/vhs"
	"github.com/christianmz565/lab-report/internal/services"
	"github.com/spf13/cobra"
)

func newCaptureCmd() *cobra.Command {
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

			fs := osfs.New()
			renderer := vhs.New()
			cfg := config.New()

			svc := services.NewCaptureService(renderer, fs, cfg)
			return svc.Execute(cmd.Context(), args[0])
		},
	}

	return cmd
}
