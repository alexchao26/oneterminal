package yaml

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/alexchao26/oneterminal/pkg/monitor"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// yaml is an internal package that will parse a user's
// yaml files with configs in ~/.config/oneterminal
// the configs require certain parameters that will
// panic the application if not found

// OneTerminalConfig of all the fields from a yaml config
type OneTerminalConfig struct {
	Name     string    `yaml:"name"`
	Shell    string    `yaml:"shell"`
	Short    string    `yaml:"short"`
	Long     string    `yaml:"long,omitempty"`
	CmdDir   string    `yaml:"directory,omitempty"`
	Commands []Command `yaml:"commands"`
}

// Command is what will run in one terminal "window"/tab
type Command struct {
	Text    string `yaml:"text"`
	Silence bool   `yaml:"silence"`
}

// ParseAndAddToRoot will be invoked when oneterminal starts
// to parse all the config files from ~/.config/oneterminal/*.yml
// and add those commands to the root command
func ParseAndAddToRoot(rootCmd *cobra.Command) {
	// Parse all command configurations
	cmdConfigs, err := ParseConfigs()
	if err != nil {
		panic(fmt.Sprintf("Reading configs %s", err))
	}

	// disallow commands with the same name
	allNames := make(map[string]bool)

	for _, config := range cmdConfigs {
		if allNames[config.Name] {
			panic(fmt.Sprintf("Multiple commands have the same name %s", config.Name))
		}
		allNames[config.Name] = true
		if config.Name == "example" {
			panic("The command name \"example\" is reserved :(")
		}

		// create the final cobra command and add it to the root command
		cobraCommand := &cobra.Command{
			Use:   config.Name,
			Short: config.Short,
			Run: func(cmd *cobra.Command, args []string) {
				// Setup coordinator and its commands
				coordinator := monitor.NewCoordinator()

				for _, cmd := range config.Commands {
					monitoredCmd := monitor.NewMonitoredCmd(cmd.Text)
					if config.Shell == "bash" {
						monitoredCmd = monitor.BashShell(monitoredCmd)
					}
					if config.CmdDir != "" {
						monitoredCmd = monitor.SetCmdDir(config.CmdDir)(monitoredCmd)
					}
					if cmd.Silence {
						monitoredCmd = monitor.SilenceOutput(monitoredCmd)
					}

					coordinator.AddCommands(monitoredCmd)
				}

				coordinator.RunCommands()
			},
		}
		if config.Long != "" {
			cobraCommand.Long = config.Long
		}
		rootCmd.AddCommand(cobraCommand)
	}
}

// ParseConfigs will parse each file in ~/.config/oneterminal
// into a slice.
// Configs are expected to have
func ParseConfigs() ([]OneTerminalConfig, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	// Unmarshal all values from configDir
	var allConfigs []OneTerminalConfig
	files, err := ioutil.ReadDir(configDir)
	if err != nil {
		return nil, errors.Wrap(err, "reading from config directory")
	}

	for _, f := range files {
		filename := fmt.Sprintf("%s/%s", configDir, f.Name())
		bytes, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, errors.Wrapf(err, "reading file %s", filename)
		}
		var oneTermConfig OneTerminalConfig
		err = yaml.Unmarshal(bytes, &oneTermConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "unmarshalling file %s", filename)
		}
		allConfigs = append(allConfigs, oneTermConfig)
	}

	return allConfigs, nil
}

// GetConfigDir returns the path to the config directory
// it should be ~/.config/oneterminal
// The directory will be made if it does not exist
func GetConfigDir() (string, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "finding home directory")
	}

	oneTermConfigDir := filepath.Join(homedir, ".config/oneterminal")

	if err := os.MkdirAll(oneTermConfigDir, os.ModePerm); err != nil {
		return "", err
	}
	return oneTermConfigDir, nil
}

// MakeExampleConfigFromStruct will generate an example config file in the
// ~/.config/oneterminal directory all required fields
// it uses the struct
func MakeExampleConfigFromStruct() error {
	oneTermConfigDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	exampleConfig := OneTerminalConfig{
		Name:  "somename",
		Shell: "zsh",
		Short: "An example command that says hello twice",
		Long: `A very polite shell command that says
hello to you multiple times. 
	
Some say you can hear it from space.`,
		CmdDir: "$HOME/go",
		Commands: []Command{
			{"echo hello from window 1", false},
			{"echo hello from window 2", false},
			{"echo hello from window 3", true},
		},
	}

	bytes, err := yaml.Marshal(exampleConfig)
	if err != nil {
		return errors.Wrap(err, "marshalling yaml")
	}

	// write to a file
	err = ioutil.WriteFile(filepath.Join(oneTermConfigDir, "generated-example.yml"), bytes, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "writing to example config file")
	}

	return nil
}

// MakeExampleConfigFromStructWithInstructions writes an example oneterminal yaml config
// to ~/.config/oneterminal/example.yml with helpful comments
func MakeExampleConfigFromStructWithInstructions() error {
	oneTermConfigDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(
		oneTermConfigDir+"/example.yml",
		[]byte(`# The name of the command, it cannot have special characters
name: somename

# shell to use, zsh and bash are supported
shell: zsh

# a short description of what this command does
short: an example command that says hello twice
# OPTIONAL: longer description of what this command does
long: Optional longer description

# OPTIONAL: directory to run the command from, use $HOME instead of ~
# this path will be expanded via os.ExpandEnv
directory: $HOME/go

# commands contain the text (the command to run, will be expanded via os.ExpandEnv)
# and an optional silence boolean, if true will silence that command's output
commands:
- text: echo hello from window 1
  silence: false
- text: echo hello from window 2
  silence: false
- text: echo they silenced me :'(
  silence: true
`),
		os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "writing to example config file")
	}

	return nil
}
