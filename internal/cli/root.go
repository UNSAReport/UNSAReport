package cli

import (
	"fmt"
	"os"

	"github.com/christianmz565/lab-report/internal/dependencies"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "lab-report",
	Short:         "Lab report template CLI",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return dependencies.CheckAll()
	},
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
		return fmt.Errorf("no command provided (try --help)")
	}
}
