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
flag is used. A backup is created before applying updates, which can be restored with --rollback.`,
		Example: `  # Update the current project interactively
  unsarep update

  # Force update all files without prompting
  unsarep update --force

  # Update a specific session in a multi-lab repository
  unsarep update l1

  # Update from a local directory
  unsarep update --local ./my-templates

  # Rollback the last update
  unsarep update --rollback`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opt.Rollback {
				fs := osfs.New()
				cfg := config.New()
				rb := services.NewRollbackService(fs, cfg)

				cwd, err := fs.Getwd()
				if err != nil {
					return err
				}
				projectRoot, _, ok, err := cfg.FindProjectRoot(cwd)
				if err != nil {
					return err
				}
				if !ok {
					projectRoot = cwd
				}

				return rb.Rollback(projectRoot)
			}

			if len(args) > 1 {
				return cmd.Help()
			}
			if len(args) == 1 {
				opt.Session = args[0]
			}

			fs := osfs.New()
			fetcher := github.New()
			cfg := config.New()
			reg := registry.NewRemote(fetcher)
			compReg := registry.NewComponentRegistry(fetcher)
			compSvc := services.NewComponentService(fetcher, fs, cfg, compReg)

			svc := services.NewUpdateService(fetcher, fs, cfg, reg, compSvc)
			return svc.Execute(cmd.Context(), opt)
		},
	}

	cmd.Flags().StringVar(&opt.Dest, "dest", "", "Destination directory (default: current working directory)")
	cmd.Flags().BoolVarP(&opt.Force, "force", "f", false, "Apply all updates without prompting")
	cmd.Flags().StringVar(&opt.Local, "local", "", "Local directory containing template files to update from")
	cmd.Flags().BoolVar(&opt.Rollback, "rollback", false, "Rollback the last update using the backup")

	return cmd
}
