package yaml

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/alexchao26/oneterminal/internal/monitor"
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
	Name        string   `yaml:"name"`
	Command     string   `yaml:"command"`
	CmdDir      string   `yaml:"directory,omitempty"`
	Silence     bool     `yaml:"silence,omitempty"`
	ReadyRegexp string   `yaml:"ready-regexp,omitempty"`
	DependsOn   []string `yaml:"depends-on,omitempty"`
}

var reservedNames = map[string]bool{
	"completion": true,
	"example":    true,
	"help":       true,
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

	for _, configPointer := range cmdConfigs {
		// this assignment to config is needed because ranging for loop assign a
		// pointer that iterates thorugh a slice, i.e. all commands would end up
		// being overwritten with the last config/element in the slice
		config := configPointer
		if allNames[config.Name] {
			panic(fmt.Sprintf("Multiple commands have the same name %s", config.Name))
		}
		allNames[config.Name] = true

		if reservedNames[config.Name] {
			panic(fmt.Sprintf("The command name %q is reserved :(", config.Name))
		}

		// create the final cobra command and add it to the root command
		cobraCommand := &cobra.Command{
			Use:   config.Name,
			Short: config.Short,
			Run: func(cmd *cobra.Command, args []string) {
				// Setup Orchestrator and its commands
				Orchestrator := monitor.NewOrchestrator()

				for _, cmd := range config.Commands {
					var options []func(monitor.MonitoredCmd) monitor.MonitoredCmd
					if cmd.Name != "" {
						options = append(options, monitor.SetCmdName(cmd.Name))
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

					monitoredCmd := monitor.NewMonitoredCmd(cmd.Command, options...)

					Orchestrator.AddCommands(&monitoredCmd)
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
				Name:        "greeter-1",
				Command:     "echo hello from window 1",
				CmdDir:      "$HOME/go",
				Silence:     false,
				ReadyRegexp: "",
			}, {
				Name:        "greeter-2",
				Command:     "echo hello from window 2",
				Silence:     false,
				ReadyRegexp: "window [0-9]",
				DependsOn:   []string{"greeter-1"},
			}, {
				Name:    "",
				Command: "echo I am silent",
				Silence: true,
			},
		},
	}

	bytes, err := yaml.Marshal(exampleConfig)
	if err != nil {
		return errors.Wrap(err, "marshalling yaml")
	}

	// write to a file
	err = ioutil.WriteFile(oneTermConfigDir+"/generated-example.yml", bytes, os.ModePerm)
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
		[]byte(`# The name of the command. Alphanumeric, dash and hypens are accepted
name: somename

# shell to use (zsh|bash), defaults to zsh
shell: zsh

# a short description of what this command does
short: an example command that says hello twice
# OPTIONAL: longer description of what this command does
long: Optional longer description

# An array of commands, each command consists of:
#   1. command {string}: the command to run directly in a shell
#   2. name {string default: ""}: used to prefix each line of this command's
#        output AND for other commands to list dependencies
#        NOTE: an empty string is a valid name and is useful for things like
#           vault which write to stdout in small chunks
#   3. directory {string, default: $HOME}: what directory to run the command in
#        NOTE: use $HOME, not ~. This strings gets passed through os.ExpandEnv
#   4. silence {boolean, default: false}, silence this command's output?
#   5. depends-on {[]string}: which (names of) commands to wait for
#   6. ready-regexp {string, optional}: a regular expression that the outputs
#        must match for this command to be considered "ready" and for its
#        dependants to begin running
commands:
- name: greeter-1
  command: echo hello from window 1
  directory: $HOME/go
  ready-regexp: "window [0-9]"
  silence: false
- name: greeter-2
  command: echo hello from window 2
  silence: false
  depends-on:
  - greeter-1
- name: ""
  command: echo "they silenced me :'("
  silence: true
`),
		os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "writing to example config file")
	}

	return nil
}
