package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for UNSAReport.

To load completions:

Bash:
  $ source <(unsarep completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ unsarep completion bash > /etc/bash_completion.d/unsarep
  # macOS:
  $ unsarep completion bash > $(brew --prefix)/etc/bash_completion.d/unsarep

Zsh:
  # If shell completion is not already enabled, enable it:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  # To load completions for each session, execute once:
  $ unsarep completion zsh > "${fpath[1]}/_unsarep"

Fish:
  $ unsarep completion fish | source
  # To load completions for each session, execute once:
  $ unsarep completion fish > ~/.config/fish/completions/unsarep.fish

PowerShell:
  PS> unsarep completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, add output to your profile.`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletionV2(cmd.OutOrStdout(), true)
			case "zsh":
				return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}

	return cmd
}
