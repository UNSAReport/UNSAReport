package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/samber/oops"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	rootCmd = &cobra.Command{
		Use:   "unsarep",
		Short: "UNSAReport template CLI",
		Long: `UNSAReport template CLI is a tool to manage and automate lab report creation.
It helps you scaffold new projects, update template files, capture terminal output,
and compile everything into a submission-ready PDF and source code bundle.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig()
		},
	}
	debugMode bool
)

func Execute() {
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("{{.Name}} {{.Version}}\n")

	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug output with stack traces")

	if err := rootCmd.Execute(); err != nil {
		if debugMode {
			debugPrintError(err)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func debugPrintError(err error) {
	if oopsErr, ok := oops.AsOops(err); ok {
		slog.Error("error",
			"error", oopsErr.Error(),
			"stacktrace", oopsErr.StackFrames(),
		)
	} else {
		slog.Error("error", "error", err)
	}
}

func initConfig() error {
	viper.SetEnvPrefix("UNSAREP")
	if err := viper.BindEnv("dest"); err != nil {
		return fmt.Errorf("bind env DEST: %w", err)
	}
	if err := viper.BindEnv("session"); err != nil {
		return fmt.Errorf("bind env SESSION: %w", err)
	}
	if err := viper.BindEnv("local"); err != nil {
		return fmt.Errorf("bind env LOCAL: %w", err)
	}
	if err := viper.BindEnv("freeze_flags", "UNSAREP_FREEZE_FLAGS"); err != nil {
		return fmt.Errorf("bind env FREEZE_FLAGS: %w", err)
	}
	return nil
}

func init() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	rootCmd.AddCommand(
		newInstallCmd(),
		newUpdateCmd(),
		newPrepareCmd(),
		newCaptureCmd(),
		newComponentCmd(),
		newCompletionCmd(),
	)

	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	}
}
