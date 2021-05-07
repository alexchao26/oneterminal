package cli

import (
	"fmt"

	"github.com/alexchao26/oneterminal/internal/yaml"
	"github.com/spf13/cobra"
)

// ExampleCmd makes an example oneterminal config file
// in the ~/.config/oneterminal directory
var ExampleCmd = &cobra.Command{
	Use:   "example",
	Short: "Makes a demo oneterminal config in ~/.config/oneterminal",
	Run: func(cmd *cobra.Command, args []string) {
		if err := yaml.WriteExampleConfig("example.yml"); err != nil {
			panic(fmt.Sprintf("Error generating example config :( %s", err))
		}
		fmt.Println("Example file generated at ~/.config/oneterminal/example.yml")
	},
}
