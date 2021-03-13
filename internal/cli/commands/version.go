package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "v0.3.0"

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Version of oneterminal",
	Args:  cobra.ExactValidArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("oneterminal", version)
	},
}
