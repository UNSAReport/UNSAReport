package cmd

import (
	"github.com/UNSAReport/UNSAReport/internal/adapters/config"
	"github.com/UNSAReport/UNSAReport/internal/adapters/github"
	"github.com/UNSAReport/UNSAReport/internal/adapters/osfs"
	"github.com/UNSAReport/UNSAReport/internal/adapters/registry"
	"github.com/UNSAReport/UNSAReport/internal/services"
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
  unsarep update

  # Force update all files without prompting
  unsarep update --force

  # Update a specific session in a multi-lab repository
  unsarep update l1

  # Update from a local directory
  unsarep update --local ./my-templates`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return cmd.Help()
			}
			if len(args) == 1 {
				opt.Session = args[0]
			}

			fs := osfs.New()
			fetcher := github.New()
			cfg := config.New()
			reg := registry.NewRemote(opt.Repo, opt.Ref, fetcher)

			svc := services.NewUpdateService(fetcher, fs, cfg, reg)
			return svc.Execute(cmd.Context(), opt)
		},
	}

	cmd.Flags().StringVar(&opt.Dest, "dest", "", "Destination directory (default: current working directory)")
	cmd.Flags().BoolVarP(&opt.Force, "force", "f", false, "Apply all updates without prompting")
	cmd.Flags().StringVar(&opt.Repo, "repo", "UNSAReport/templates", "GitHub repo to fetch templates from (owner/repo)")
	cmd.Flags().StringVar(&opt.Ref, "ref", "main", "Git ref to fetch templates from")
	cmd.Flags().StringVar(&opt.Local, "local", "", "Local directory containing template files to update from")

	cmd.MarkFlagsMutuallyExclusive("local", "repo")
	cmd.MarkFlagsMutuallyExclusive("local", "ref")

	return cmd
}
