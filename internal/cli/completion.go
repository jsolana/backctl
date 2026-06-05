package cli

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate a shell completion script for backctl.

To load completions:

  bash:
    source <(backctl completion bash)

    # To persist across sessions (Linux):
    backctl completion bash > /etc/bash_completion.d/backctl

    # To persist across sessions (macOS with Homebrew):
    backctl completion bash > $(brew --prefix)/etc/bash_completion.d/backctl

  zsh:
    # If shell completion is not already enabled, add to ~/.zshrc:
    autoload -U compinit; compinit

    source <(backctl completion zsh)

    # To persist across sessions:
    backctl completion zsh > "${fpath[1]}/_backctl"

  fish:
    backctl completion fish | source

    # To persist across sessions:
    backctl completion fish > ~/.config/fish/completions/backctl.fish

  powershell:
    backctl completion powershell | Out-String | Invoke-Expression

    # To persist across sessions, add the output to your profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
	return cmd
}
