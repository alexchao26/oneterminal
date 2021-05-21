package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/alexchao26/oneterminal/internal/yaml"
	"github.com/spf13/cobra"
)

func makeListCmd(allConfigs []yaml.OneTerminalConfig) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List only configured commands",
		Long: `Lists the names of all commands configured in ~/.config/oneterminal/{file}.yaml

Excludes built in commands.`,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("Configured commands (runable via `oneterminal <name>`)")
			fmt.Println()
			w := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)
			for _, config := range allConfigs {
				fmt.Fprintf(w, "%s:\t%s\n", config.Name, config.Short)
			}
			w.Flush()
		},
	}
}
