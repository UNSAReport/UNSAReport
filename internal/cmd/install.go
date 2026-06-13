package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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

  # Install in a specific directory
  unsarep install lab --dest ./my-reports

  # Install a multi-lab template
  unsarep install multi-lab`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return cmd.Help()
			}

			var templateName string
			if len(args) == 1 {
				templateName = args[0]
			} else {
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

			registryPath := findTemplatesDir()
			reg := registry.New(registryPath)

			svc := services.NewInstallService(fetcher, fs, cfg, reg)
			return svc.Execute(cmd.Context(), opt)
		},
	}

	cmd.Flags().StringVar(&opt.Dest, "dest", "", "Destination directory (default: current working directory)")
	cmd.Flags().StringVar(&opt.Session, "session", "", "Session/Lab name for per-lab installation in multi-lab templates")
	cmd.Flags().StringVar(&opt.Repo, "repo", "UNSAReport/templates", "GitHub repo to fetch templates from (owner/repo)")
	cmd.Flags().StringVar(&opt.Ref, "ref", "main", "Git ref to fetch templates from")

	return cmd
}

func pickTemplate() (string, error) {
	registryPath := findTemplatesDir()
	reg := registry.New(registryPath)

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

	form := huh.NewSelect[string]().
		Title("Select a template").
		Options(options...).
		Value(&selected)

	if err := form.Run(); err != nil {
		return "", fmt.Errorf("template selection cancelled: %w", err)
	}

	return selected, nil
}

func findTemplatesDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "templates"
	}

	for dir := cwd; dir != "/"; dir = filepath.Dir(dir) {
		templatesDir := filepath.Join(dir, "templates")
		if info, err := os.Stat(templatesDir); err == nil && info.IsDir() {
			return templatesDir
		}
	}

	return filepath.Join(cwd, "templates")
}
