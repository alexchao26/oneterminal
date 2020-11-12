package cli

import (
	"github.com/alexchao26/oneterminal/pkg/monitor"

	"github.com/spf13/cobra"
)

// DemoCmd was used in creating the monitor pkg
// It is left here as an artifact and can be used to
// test changes to the root command without using a
// yml config. Don't forget to Add DemoCmd to rootCmd
var DemoCmd = &cobra.Command{
	Use:   "demo",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		m1 := monitor.NewMonitoredCmd(
			"go run internal/demo/ticker-service/ticker.go",
			monitor.SetCmdDir("/Users/chao/go/src/github.com/alexchao26/oneterminal"))
		m2 := monitor.NewMonitoredCmd(
			"go run internal/demo/ticker-service/ticker.go",
			monitor.SetCmdDir("/Users/chao/go/src/github.com/alexchao26/oneterminal"),
			monitor.SilenceOutput)

		orchestrator := monitor.NewOrchestrator()
		orchestrator.AddCommands(m1)
		orchestrator.AddCommands(m2)
		orchestrator.RunCommands()
	},
}
