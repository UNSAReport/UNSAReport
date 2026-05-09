package cmd

import (
	"github.com/christianmz565/lab-report/internal/adapters/config"
	"github.com/christianmz565/lab-report/internal/adapters/osfs"
	"github.com/christianmz565/lab-report/internal/adapters/typst"
	"github.com/christianmz565/lab-report/internal/adapters/zipper"
	"github.com/christianmz565/lab-report/internal/services"
	"github.com/spf13/cobra"
)

func newPrepareCmd() *cobra.Command {
	opt := services.PrepareOptions{}

	cmd := &cobra.Command{
		Use:   "prepare [lab-dir]",
		Short: "Compile the report and zip the source code",
		Long: `Compile the Typst report into a PDF and create a ZIP archive of the source code.

This command reads 'report.typ' to extract project metadata (course, lab, authors)
and uses it to generate properly named files in a 'submission' folder.`,
		Example: `  # Prepare submission in a single-lab project
  lab-report prepare

  # Prepare submission in a multi-lab project (from root)
  lab-report prepare l1

  # Force reconfiguration of file naming template
  lab-report prepare --configure`,
		RunE: func(cmd *cobra.Command, args []string) error {
			labDir := ""
			if len(args) > 0 {
				labDir = args[0]
			}
			if len(args) > 1 {
				return cmd.Help()
			}

			fs := osfs.New()
			compiler := typst.New()
			archiver := zipper.New()
			cfg := config.New()

			svc := services.NewPrepareService(compiler, archiver, fs, cfg)
			return svc.Execute(cmd.Context(), opt, labDir)
		},
	}

	cmd.Flags().BoolVarP(&opt.Configure, "configure", "c", false, "Force reconfiguration of submission file names")

	return cmd
}
