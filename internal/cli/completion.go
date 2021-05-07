package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// CompletionCmd returns a string that can be piped to add bash/zsh completions
var CompletionCmd = &cobra.Command{
	Use:   "completion [bash|zsh]",
	Short: "Generate completion script",
	Long: `To load completions:
Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To persist completions, execute once:
$ oneterminal completion zsh > "${fpath[1]}/_oneterminal"

Bash:

$ source <(oneterminal completion bash)

# To persist completions, execute once:
Linux:
  $ oneterminal completion bash > /etc/bash_completion.d/oneterminal
MacOS:
  $ oneterminal completion bash > /usr/local/etc/bash_completion.d/oneterminal
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		}
	},
}
