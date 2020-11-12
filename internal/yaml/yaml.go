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
	Commands []Command `yaml:"commands"`
}

// Command is what will run in one terminal "window"/tab
type Command struct {
	Text    string `yaml:"text"`
	CmdDir  string `yaml:"directory,omitempty"`
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
				// Setup Orchestrator and its commands
				Orchestrator := monitor.NewOrchestrator()

				for i, cmd := range config.Commands {
					prefix := fmt.Sprintf("[%02d]", i)
					monitoredCmd := monitor.NewMonitoredCmd(cmd.Text, prefix)
					if config.Shell == "bash" {
						monitoredCmd = monitor.BashShell(monitoredCmd)
					}
					if cmd.CmdDir != "" {
						monitoredCmd = monitor.SetCmdDir(cmd.CmdDir)(monitoredCmd)
					}
					if cmd.Silence {
						monitoredCmd = monitor.SilenceOutput(monitoredCmd)
					}

					Orchestrator.AddCommands(monitoredCmd)
				}

				Orchestrator.RunCommands()
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
		Commands: []Command{
			{
				Text:    "echo hello from window 1",
				CmdDir:  "$HOME/go",
				Silence: false,
			}, {
				Text:    "echo hello from window 2",
				Silence: false,
			}, {
				Text:    "echo they silenced me :(",
				Silence: true,
			},
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

# commands are made of
#   1. text string (the command to run, will be expanded via os.ExpandEnv)
#   2. directory string (optional), what directory to run the command from
#      NOTE: use $HOME, not ~. This strings gets passed through os.ExpandEnv
#   3. silence boolean (optional: default false), if true will silence that command's output
commands:
- text: echo hello from window 1
  directory: $HOME/go
  silence: false
- text: echo hello from window 2
  silence: false
- text: echo "they silenced me :'("
  silence: true
`),
		os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "writing to example config file")
	}

	return nil
}
