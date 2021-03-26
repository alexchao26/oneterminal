package cli

import (
	"context"
	"fmt"

	"github.com/alexchao26/oneterminal/cmdsync"
	"github.com/alexchao26/oneterminal/internal/yaml"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Init creates the root command by parsing all yaml configs from the
// ~/.config/oneterminal directory and adding them to the root command.
//
// All commands will be accessible via oneterminal <command-name>
func Init(version string) (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use:   "oneterminal",
		Short: "oneterminal replaces your multi-tab terminal window setup",
		Long: `oneterminal makes shell scripts configurable via yaml.
It strives to reduce the number of terminal windows
that need to be open.

Config files live in ~/.config/oneterminal
Run "oneterminal example" to generate an example config file`,
	}

	allConfigs, err := yaml.ParseAllConfigs()
	if err != nil {
		return nil, errors.Wrap(err, "parsing yml configs")
	}

	if err := yaml.HasNameCollisions(allConfigs); err != nil {
		return nil, err
	}

	generatedCommands := makeCommands(allConfigs)

	for _, cmd := range generatedCommands {
		rootCmd.AddCommand(cmd)
	}

	rootCmd.AddCommand(ExampleCmd)
	rootCmd.AddCommand(CompletionCmd)
	rootCmd.AddCommand(makeUpdateCmd(version))
	rootCmd.AddCommand(makeVersionCmd(version))

	return rootCmd, nil
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

	for _, config := range configs {
		config := config

		// create the final cobra command and add it to the root command
		cobraCommand := &cobra.Command{
			Use:   config.Name,
			Short: config.Short,
			Long:  config.Long,
			Run: func(cmd *cobra.Command, args []string) {
				group := cmdsync.NewGroup()
				var colorIndex int

				for _, cmd := range config.Commands {
					var options []cmdsync.CmdOption
					if cmd.Name != "" {
						options = append(options, cmdsync.CmdName(cmd.Name))
						options = append(options, cmdsync.SetColor(ansiColors[colorIndex%len(ansiColors)]))
						colorIndex++
					}
					if cmd.CmdDir != "" {
						options = append(options, cmdsync.CmdDir(cmd.CmdDir))
					}
					if cmd.Silence {
						options = append(options, cmdsync.SilenceOutput())
					}
					if cmd.ReadyRegexp != "" {
						options = append(options, cmdsync.ReadyPattern(cmd.ReadyRegexp))
					}
					if len(cmd.DependsOn) != 0 {
						options = append(options, cmdsync.DependsOn(cmd.DependsOn))
					}
					if cmd.Environment != nil {
						options = append(options, cmdsync.Environment(cmd.Environment))
					}

					c, err := cmdsync.NewCmd(config.Shell, cmd.Command, options...)
					if err != nil {
						panic(fmt.Sprintf("error making command %q: %v", cmd.Name, err))
					}

					group.AddCommands(c)
				}

				err := group.Run(context.Background())
				if err != nil {
					fmt.Printf("running %q: %v\n", config.Name, err)
				}
			},
		}

		cobraCommands = append(cobraCommands, cobraCommand)
	}

	return cobraCommands
}
