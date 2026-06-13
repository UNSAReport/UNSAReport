package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "unsarep",
	Short: "UNSReport template CLI",
	Long: `UNSReport template CLI is a tool to manage and automate lab report creation.
It helps you scaffold new projects, update template files, capture terminal output,
and compile everything into a submission-ready PDF and source code bundle.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("{{.Name}} {{.Version}}\n")

	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	rootCmd.AddCommand(
		newInstallCmd(),
		newUpdateCmd(),
		newPrepareCmd(),
		newCaptureCmd(),
	)

	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	}
}
