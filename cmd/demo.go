package cmd

import (
	"github.com/alexchao26/oneterminal/pkg/monitor-cmd"

	"github.com/spf13/cobra"
)

// demoCmd represents the demo command
var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		m1 := monitor.NewMonitoredCmd(
			"go run internal/demo-services/ticker.go",
			monitor.SetCmdDir("/Users/chao/go/src/github.com/alexchao26/oneterminal"))
		m2 := monitor.NewMonitoredCmd(
			"go run internal/demo-services/ticker.go",
			monitor.SetCmdDir("/Users/chao/go/src/github.com/alexchao26/oneterminal"),
			monitor.SilenceOutput)

		coord := monitor.NewCoordinator()
		coord.AddCommands(m1)
		coord.AddCommands(m2)
		coord.RunCommands()
	},
}

func init() {
	rootCmd.AddCommand(demoCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// demoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// demoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
