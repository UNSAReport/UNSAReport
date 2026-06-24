package cmd

import (
	"fmt"

	"github.com/UNSAReport/UNSAReport/internal/adapters/config"
	"github.com/UNSAReport/UNSAReport/internal/adapters/github"
	"github.com/UNSAReport/UNSAReport/internal/adapters/osfs"
	"github.com/UNSAReport/UNSAReport/internal/adapters/registry"
	"github.com/UNSAReport/UNSAReport/internal/services"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	opt := services.InstallOptions{}

	cmd := &cobra.Command{
		Use:   "install [template]",
		Short: "Install the template into a destination directory",
		Long: `Install the lab report template files into a specified directory.

This command downloads the latest template files (Typst files, bibliography, images, etc.)
from the repository and initializes a 'unsareport.json' configuration file.

If no template is specified, an interactive picker will be shown.`,
		Example: `  # Install with interactive template picker
  unsarep install

  # Install a specific template
  unsarep install lab

  # Install a specific version
  unsarep install lab@1.0.0

  # Install with version range
  unsarep install lab@^1.0.0

  # Install in a specific directory
  unsarep install lab --dest ./my-reports

  # Install a multi-lab template
  unsarep install multi-lab

  # Install from a local directory
  unsarep install --local ./my-templates`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return cmd.Help()
			}

			var templateName string
			switch {
			case len(args) == 1:
				templateName = args[0]
			case opt.Local != "":
				selected, err := pickLocalTemplate(opt.Local)
				if err != nil {
					return err
				}
				templateName = selected
			default:
				selected, err := pickTemplate()
				if err != nil {
					return err
				}
				templateName = selected
			}

			opt.Template = templateName

			fs := osfs.New()
			fetcher := github.New()
			cfg := config.New()

			reg := registry.NewRemote(fetcher)
			compReg := registry.NewComponentRegistry(fetcher)
			compSvc := services.NewComponentService(
				services.WithComponentFetcher(fetcher),
				services.WithComponentFS(fs),
				services.WithComponentConfig(cfg),
				services.WithComponentRegistry(compReg),
				services.WithComponentStdout(cmd.OutOrStdout()),
				services.WithComponentStderr(cmd.ErrOrStderr()),
			)

			svc := services.NewInstallService(
				services.WithInstallFetcher(fetcher),
				services.WithInstallFS(fs),
				services.WithInstallConfig(cfg),
				services.WithInstallRegistry(reg),
				services.WithInstallComponentService(compSvc),
				services.WithInstallStdout(cmd.OutOrStdout()),
				services.WithInstallStderr(cmd.ErrOrStderr()),
			)
			return svc.Execute(cmd.Context(), opt)
		},
	}

	cmd.Flags().StringVar(&opt.Dest, "dest", "", "Destination directory (default: current working directory)")
	cmd.Flags().StringVar(&opt.Session, "session", "", "Session/Lab name for per-lab installation in multi-lab templates")
	cmd.Flags().StringVar(&opt.Local, "local", "", "Local directory containing template files to install from")

	return cmd
}

func pickTemplate() (string, error) {
	fetcher := github.New()
	reg := registry.NewRemote(fetcher)

	templates, err := reg.ListTemplates()
	if err != nil {
		return "", fmt.Errorf("failed to list templates: %w", err)
	}

	if len(templates) == 0 {
		return "", fmt.Errorf("no templates found")
	}

	var selected string
	options := make([]huh.Option[string], len(templates))
	for i, t := range templates {
		options[i] = huh.NewOption(t.Name, t.Name)
	}

	descriptionMap := make(map[string]string, len(templates))
	for _, t := range templates {
		descriptionMap[t.Name] = t.Description
	}

	form := huh.NewSelect[string]().
		Title("Select a template").
		Options(options...).
		Value(&selected).
		DescriptionFunc(func() string {
			return descriptionMap[selected]
		}, &selected)

	if err := form.Run(); err != nil {
		return "", fmt.Errorf("template selection cancelled: %w", err)
	}

	return selected, nil
}

func pickLocalTemplate(localDir string) (string, error) {
	reg := registry.NewLocal(localDir)

	templates, err := reg.ListTemplates()
	if err != nil {
		return "", fmt.Errorf("failed to list templates: %w", err)
	}

	if len(templates) == 0 {
		return "", fmt.Errorf("no templates found in %s", localDir)
	}

	if len(templates) == 1 {
		return templates[0].Name, nil
	}

	var selected string
	options := make([]huh.Option[string], len(templates))
	for i, t := range templates {
		options[i] = huh.NewOption(t.Name, t.Name)
	}

	form := huh.NewSelect[string]().
		Title("Select a template").
		Options(options...).
		Value(&selected)

	if err := form.Run(); err != nil {
		return "", fmt.Errorf("template selection cancelled: %w", err)
	}

	return selected, nil
}
