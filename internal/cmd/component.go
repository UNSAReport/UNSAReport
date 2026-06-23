package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/UNSAReport/UNSAReport/internal/adapters/config"
	"github.com/UNSAReport/UNSAReport/internal/adapters/github"
	"github.com/UNSAReport/UNSAReport/internal/adapters/osfs"
	"github.com/UNSAReport/UNSAReport/internal/adapters/registry"
	"github.com/UNSAReport/UNSAReport/internal/services"
	"github.com/spf13/cobra"
)

func newComponentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "component",
		Short: "Manage components",
		Long:  `Add, remove, list, and update Typst components in your project.`,
	}

	cmd.AddCommand(
		newComponentListCmd(),
		newComponentAddCmd(),
		newComponentRemoveCmd(),
		newComponentUpdateCmd(),
	)

	return cmd
}

func newComponentListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available components",
		RunE: func(cmd *cobra.Command, args []string) error {
			fs := osfs.New()
			fetcher := github.New()
			cfg := config.New()
			compReg := registry.NewComponentRegistry(fetcher)

			svc := services.NewComponentService(fetcher, fs, cfg, compReg)
			return svc.List(cmd.Context())
		},
	}

	return cmd
}

func newComponentAddCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "add <name>[@range]",
		Short: "Add a component to the project",
		Example: `  # Add with latest version
  unsarep component add code-block

  # Add with specific version range
  unsarep component add code-block@^1.0.0

  # Force overwrite local modifications
  unsarep component add code-block --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, rangeSpec := parseComponentArg(args[0])

			fs := osfs.New()
			fetcher := github.New()
			cfg := config.New()
			compReg := registry.NewComponentRegistry(fetcher)

			svc := services.NewComponentService(fetcher, fs, cfg, compReg)
			if err := svc.Add(cmd.Context(), name, rangeSpec, force); err != nil {
				return err
			}

			fmt.Fprintf(os.Stdout, "Added: %s\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force overwrite local modifications")

	return cmd
}

func newComponentRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <name>",
		Short:   "Remove a component from the project",
		Example: `  unsarep component remove code-block`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fs := osfs.New()
			fetcher := github.New()
			cfg := config.New()
			compReg := registry.NewComponentRegistry(fetcher)

			svc := services.NewComponentService(fetcher, fs, cfg, compReg)
			if err := svc.Remove(cmd.Context(), args[0]); err != nil {
				return err
			}

			fmt.Fprintf(os.Stdout, "Removed: %s\n", args[0])
			return nil
		},
	}

	return cmd
}

func newComponentUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [name]",
		Short: "Update components to latest versions",
		Example: `  # Update all components
  unsarep component update

  # Update a specific component
  unsarep component update code-block`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			}

			fs := osfs.New()
			fetcher := github.New()
			cfg := config.New()
			compReg := registry.NewComponentRegistry(fetcher)

			svc := services.NewComponentService(fetcher, fs, cfg, compReg)
			return svc.Update(cmd.Context(), name)
		},
	}

	return cmd
}

func parseComponentArg(arg string) (name, rangeSpec string) {
	if i := strings.Index(arg, "@"); i != -1 {
		return arg[:i], arg[i+1:]
	}
	return arg, "latest"
}
