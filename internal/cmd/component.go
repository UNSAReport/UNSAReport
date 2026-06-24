package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Masterminds/semver/v3"
	"github.com/UNSAReport/UNSAReport/internal/adapters/config"
	"github.com/UNSAReport/UNSAReport/internal/adapters/github"
	"github.com/UNSAReport/UNSAReport/internal/adapters/osfs"
	"github.com/UNSAReport/UNSAReport/internal/adapters/registry"
	"github.com/UNSAReport/UNSAReport/internal/ports"
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

			svc := services.NewComponentService(fetcher, fs, cfg, compReg, cmd.OutOrStdout(), cmd.ErrOrStderr())
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

			svc := services.NewComponentService(fetcher, fs, cfg, compReg, cmd.OutOrStdout(), cmd.ErrOrStderr())
			if err := svc.Add(cmd.Context(), name, rangeSpec, force); err != nil {
				return err
			}

			checkTemplateCompatibility(cmd.Context(), fetcher, cfg, name)

			fmt.Fprintf(cmd.OutOrStdout(), "Added: %s\n", name)
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

			svc := services.NewComponentService(fetcher, fs, cfg, compReg, cmd.OutOrStdout(), cmd.ErrOrStderr())
			if err := svc.Remove(cmd.Context(), args[0]); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Removed: %s\n", args[0])
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

			svc := services.NewComponentService(fetcher, fs, cfg, compReg, cmd.OutOrStdout(), cmd.ErrOrStderr())
			return svc.Update(cmd.Context(), name)
		},
	}

	return cmd
}

func parseComponentArg(arg string) (name, rangeSpec string) {
	return parseArg(arg)
}

func checkTemplateCompatibility(ctx context.Context, fetcher ports.TemplateFetcher, cfgStore ports.ConfigStore, componentName string) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	_, projectCfg, ok, err := cfgStore.FindProjectRoot(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not check template compatibility: %v\n", err)
		return
	}
	if !ok || projectCfg.Template == "" {
		return
	}

	tmpl, err := registry.NewRemote(fetcher).GetTemplateVersion(projectCfg.Template, projectCfg.TemplateVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not fetch template for compatibility check: %v\n", err)
		return
	}

	manifestData, err := fetcher.FetchRaw(ctx, ports.DefaultTemplateRepo, ports.DefaultRef, tmpl.Path+"/manifest.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not fetch manifest for compatibility check: %v\n", err)
		return
	}

	var manifest struct {
		Components map[string]string `json:"components"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not parse manifest for compatibility check: %v\n", err)
		return
	}

	templateRange, exists := manifest.Components[componentName]
	if !exists {
		fmt.Fprintf(os.Stderr, "Warning: Component %s is not required by the current template\n", componentName)
		return
	}

	compReg := registry.NewComponentRegistry(fetcher)
	resolved, _, _, err := compReg.ResolveVersion(componentName, templateRange)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not verify compatibility with template requirement %s@%s\n", componentName, templateRange)
		return
	}

	componentEntry, exists := projectCfg.Components[componentName]
	if !exists {
		fmt.Fprintf(os.Stderr, "Warning: Component %s is required by the template but not installed\n", componentName)
		return
	}

	componentVersion, err := semver.NewVersion(componentEntry.Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not parse installed component version: %v\n", err)
		return
	}

	constraint, err := semver.NewConstraint(templateRange)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not parse template version constraint: %v\n", err)
		return
	}

	if ok, _ := constraint.Validate(componentVersion); !ok {
		fmt.Fprintf(os.Stderr, "Warning: Installed version %s may be incompatible with template requirement %s@%s (resolved: %s)\n",
			componentVersion, componentName, templateRange, resolved)
	}
}
