package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func makeVersionCmd(v string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Version of oneterminal",
		Args:  cobra.ExactValidArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("oneterminal v%s\n", v)
		},
	}
}
