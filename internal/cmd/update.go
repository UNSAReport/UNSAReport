package cmd

import (
	"github.com/christianmz565/lab-report/internal/adapters/config"
	"github.com/christianmz565/lab-report/internal/adapters/github"
	"github.com/christianmz565/lab-report/internal/adapters/osfs"
	"github.com/christianmz565/lab-report/internal/services"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	opt := services.UpdateOptions{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an existing template installation",
		Long: `Update the currently installed template files.

This command compares your local files against the latest versions in the repository.
It prompts you to apply changes line-by-line (using a diff view) unless the --force
flag is used.`,
		Example: `  # Update the current project interactively
  lab-report update

  # Force update all files without prompting
  lab-report update --force

  # Update a project in a different directory
  lab-report update --dest ./my-reports`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmd.Help()
			}

			fs := osfs.New()
			fetcher := github.New()
			cfg := config.New()

			svc := services.NewUpdateService(fetcher, fs, cfg)
			return svc.Execute(cmd.Context(), opt)
		},
	}

	cmd.Flags().StringVar(&opt.Dest, "dest", "", "Destination directory (default: current working directory)")
	cmd.Flags().BoolVarP(&opt.Force, "force", "f", false, "Apply all updates without prompting")
	cmd.Flags().StringVar(&opt.Repo, "repo", "christianmz565/lab-report", "GitHub repo to fetch templates from (owner/repo)")
	cmd.Flags().StringVar(&opt.Ref, "ref", "main", "Git ref to fetch templates from")

	return cmd
}
