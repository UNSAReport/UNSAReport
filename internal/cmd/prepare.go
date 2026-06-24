package cmd

import (
	"github.com/UNSAReport/UNSAReport/internal/adapters/config"
	"github.com/UNSAReport/UNSAReport/internal/adapters/osfs"
	"github.com/UNSAReport/UNSAReport/internal/adapters/typst"
	"github.com/UNSAReport/UNSAReport/internal/adapters/zipper"
	"github.com/UNSAReport/UNSAReport/internal/services"
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
  unsarep prepare

  # Prepare submission in a multi-lab project (from root)
  unsarep prepare l1

  # Force reconfiguration of file naming template
  unsarep prepare --configure`,
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

			svc := services.NewPrepareService(compiler, archiver, fs, cfg, cmd.OutOrStdout(), cmd.ErrOrStderr())
			return svc.Execute(cmd.Context(), opt, labDir)
		},
	}

	cmd.Flags().BoolVarP(&opt.Configure, "configure", "c", false, "Force reconfiguration of submission file names")

	return cmd
}
