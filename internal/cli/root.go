package cli

import (
	"fmt"
	"os"

	"github.com/alexchao26/oneterminal/internal/cli/commands"
	"github.com/alexchao26/oneterminal/internal/monitor"
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

// Execute the root command
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
	allConfigs, err := yaml.ParseAllConfigs()
	if err != nil {
		panic(err)
	}

	if yaml.HasNameCollisions(allConfigs) {
		os.Exit(1)
	}

	generatedCommands := makeCommands(allConfigs)

	for _, cmd := range generatedCommands {
		rootCmd.AddCommand(cmd)
	}

	rootCmd.AddCommand(commands.ExampleCmd)
	rootCmd.AddCommand(commands.CompletionCmd)
	rootCmd.AddCommand(commands.VersionCmd)
}

func makeCommands(configs []yaml.OneTerminalConfig) []*cobra.Command {
	ansiColors := []string{
		"\033[36;1m", // intense cyan
		"\033[32;1m", // intense green
		"\033[35;1m", // intense magenta
		"\033[34;1m", // intense blue
		"\033[33;1m", // intense yellow
		"\033[36m",   // cyan
		"\033[32m",   // green
		"\033[35m",   // magenta
		"\033[34m",   // blue
		"\033[33m",   // yellow
	}

	var cobraCommands []*cobra.Command

	for _, configPointer := range configs {
		// this assignment to config is needed because ranging for loop assign a
		// pointer that iterates thorugh a slice, i.e. all commands would end up
		// being overwritten with the last config/element in the slice
		config := configPointer

		// create the final cobra command and add it to the root command
		cobraCommand := &cobra.Command{
			Use:   config.Name,
			Short: config.Short,
			Long:  config.Long,
			Run: func(cmd *cobra.Command, args []string) {
				// Setup Orchestrator and its commands
				Orchestrator := monitor.NewOrchestrator()
				var colorIndex int

				for _, cmd := range config.Commands {
					var options []monitor.MonitoredCmdOption
					if cmd.Name != "" {
						options = append(options, monitor.SetCmdName(cmd.Name))
						options = append(options, monitor.SetColor(ansiColors[colorIndex]))
						colorIndex++
					}
					if config.Shell == "bash" {
						options = append(options, monitor.SetBashShell)
					}
					if cmd.CmdDir != "" {
						options = append(options, monitor.SetCmdDir(cmd.CmdDir))
					}
					if cmd.Silence {
						options = append(options, monitor.SetSilenceOutput)
					}
					if cmd.ReadyRegexp != "" {
						options = append(options, monitor.SetReadyPattern(cmd.ReadyRegexp))
					}
					if len(cmd.DependsOn) != 0 {
						options = append(options, monitor.SetDependsOn(cmd.DependsOn))
					}
					if cmd.Environment != nil {
						options = append(options, monitor.SetEnvironment(cmd.Environment))
					}

					monitoredCmd := monitor.NewMonitoredCmd(cmd.Command, options...)

					Orchestrator.AddCommands(monitoredCmd)
				}

				Orchestrator.RunCommands()
			},
		}

		cobraCommands = append(cobraCommands, cobraCommand)
	}

	return cobraCommands
}
