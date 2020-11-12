package cli

import (
	"fmt"
	"os"

	"github.com/alexchao26/oneterminal/internal/commands"
	"github.com/alexchao26/oneterminal/internal/yaml"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "oneterminal",
	Short: "oneterminal replaces your multi-tab terminal window setup",
	Long: `oneterminal makes shell scripts configurable via yaml.
It strives to reduce the number of terminal windows
that need to be open.

Config files live in ~/.config/oneterminal
Run oneterminal example to generate an example config file`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// On initialization, parse all yaml file configs from ~/.config/oneterminal
// directory and add them to the root command.
// All commands will be accessible via oneterminal <command-name>
// where command-name comes from each yaml file.
func init() {
	yaml.ParseAndAddToRoot(rootCmd)

	rootCmd.AddCommand(commands.ExampleCmd)
}
