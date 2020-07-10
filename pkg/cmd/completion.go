package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

$ source <(kubeconsole completion bash)

# To load completions for each session, execute once:
Linux:
  $ kubeconsole completion bash > /etc/bash_completion.d/kubeconsole
MacOS:
  $ kubeconsole completion bash > /usr/local/etc/bash_completion.d/kubeconsole

Zsh:

$ source <(kubeconsole completion zsh)

# To load completions for each session, execute once:
$ kubeconsole completion zsh > "${fpath[1]}/_kubeconsole"

Fish:

$ kubeconsole completion fish | source

# To load completions for each session, execute once:
$ kubeconsole completion fish > ~/.config/fish/completions/kubeconsole.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletion(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
