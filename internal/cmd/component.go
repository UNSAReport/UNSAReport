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
	"github.com/charmbracelet/huh"
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
	var repo, ref string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available components",
		RunE: func(cmd *cobra.Command, args []string) error {
			fs := osfs.New()
			fetcher := github.New()
			cfg := config.New()
			compReg := registry.NewComponentRegistry(repo, ref, fetcher)

			svc := services.NewComponentService(fetcher, fs, cfg, compReg)
			return svc.List(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "UNSAReport/components", "Component registry repo")
	cmd.Flags().StringVar(&ref, "ref", "main", "Git ref")

	return cmd
}

func newComponentAddCmd() *cobra.Command {
	var repo, ref string
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
			if rangeSpec == "" {
				rangeSpec = "latest"
			}

			fs := osfs.New()
			fetcher := github.New()
			cfg := config.New()
			compReg := registry.NewComponentRegistry(repo, ref, fetcher)

			svc := services.NewComponentService(fetcher, fs, cfg, compReg)
			if err := svc.Add(cmd.Context(), name, rangeSpec, force); err != nil {
				return err
			}

			fmt.Fprintf(os.Stdout, "Added: %s\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "UNSAReport/components", "Component registry repo")
	cmd.Flags().StringVar(&ref, "ref", "main", "Git ref")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force overwrite local modifications")

	return cmd
}

func newComponentRemoveCmd() *cobra.Command {
	var repo, ref string

	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a component from the project",
		Example: `  unsarep component remove code-block`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fs := osfs.New()
			fetcher := github.New()
			cfg := config.New()
			compReg := registry.NewComponentRegistry(repo, ref, fetcher)

			svc := services.NewComponentService(fetcher, fs, cfg, compReg)
			if err := svc.Remove(cmd.Context(), args[0]); err != nil {
				return err
			}

			fmt.Fprintf(os.Stdout, "Removed: %s\n", args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "UNSAReport/components", "Component registry repo")
	cmd.Flags().StringVar(&ref, "ref", "main", "Git ref")

	return cmd
}

func newComponentUpdateCmd() *cobra.Command {
	var repo, ref string

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
			compReg := registry.NewComponentRegistry(repo, ref, fetcher)

			svc := services.NewComponentService(fetcher, fs, cfg, compReg)
			return svc.Update(cmd.Context(), name)
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "UNSAReport/components", "Component registry repo")
	cmd.Flags().StringVar(&ref, "ref", "main", "Git ref")

	return cmd
}

func parseComponentArg(arg string) (name, rangeSpec string) {
	if i := strings.Index(arg, "@"); i != -1 {
		return arg[:i], arg[i+1:]
	}
	return arg, ""
}

func pickComponent(repo, ref string) (string, error) {
	fetcher := github.New()
	compReg := registry.NewComponentRegistry(repo, ref, fetcher)

	components, err := compReg.ListComponents()
	if err != nil {
		return "", fmt.Errorf("failed to list components: %w", err)
	}

	if len(components) == 0 {
		return "", fmt.Errorf("no components found")
	}

	var selected string
	options := make([]huh.Option[string], len(components))
	for i, c := range components {
		options[i] = huh.NewOption(c.Name, c.Name)
	}

	descriptionMap := make(map[string]string, len(components))
	for _, c := range components {
		descriptionMap[c.Name] = c.Description
	}

	form := huh.NewSelect[string]().
		Title("Select a component").
		Options(options...).
		Value(&selected).
		DescriptionFunc(func() string {
			return descriptionMap[selected]
		}, &selected)

	if err := form.Run(); err != nil {
		return "", fmt.Errorf("component selection cancelled: %w", err)
	}

	return selected, nil
}
