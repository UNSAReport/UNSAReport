package cmd

import (
	"github.com/christianmz565/lab-report/internal/adapters/config"
	"github.com/christianmz565/lab-report/internal/adapters/github"
	"github.com/christianmz565/lab-report/internal/adapters/osfs"
	"github.com/christianmz565/lab-report/internal/services"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	opt := services.InstallOptions{}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the template into a destination directory",
		Long: `Install the lab report template files into a specified directory.

This command downloads the latest template files (Typst files, bibliography, images, etc.)
from the repository and initializes a 'labreport.json' configuration file.`,
		Example: `  # Install in the current directory
  lab-report install

  # Install in a specific directory
  lab-report install --dest ./my-reports

  # Install with multi-lab repository structure
  lab-report install --multi`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmd.Help()
			}

			fs := osfs.New()
			fetcher := github.New()
			cfg := config.New()

			svc := services.NewInstallService(fetcher, fs, cfg)
			return svc.Execute(cmd.Context(), opt)
		},
	}

	cmd.Flags().StringVar(&opt.Dest, "dest", "", "Destination directory (default: current working directory)")
	cmd.Flags().BoolVar(&opt.Multi, "multi", false, "Install as a multi-lab repository")
	cmd.Flags().StringVar(&opt.Repo, "repo", "christianmz565/lab-report", "GitHub repo to fetch templates from (owner/repo)")
	cmd.Flags().StringVar(&opt.Ref, "ref", "main", "Git ref to fetch templates from")

	return cmd
}
