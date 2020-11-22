package yaml

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
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
	Name        string            `yaml:"name"`
	Command     string            `yaml:"command"`
	CmdDir      string            `yaml:"directory,omitempty"`
	Silence     bool              `yaml:"silence,omitempty"`
	ReadyRegexp string            `yaml:"ready-regexp,omitempty"`
	DependsOn   []string          `yaml:"depends-on,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
}

// ParseAllConfigs will parse each file in ~/.config/oneterminal
// into a slice.
// Configs are expected to have
func ParseAllConfigs() ([]OneTerminalConfig, error) {
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
		if !strings.HasSuffix(f.Name(), ".yml") {
			continue
		}

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

// HasNameCollisions ensures there are no duplicate names and doesn't use
// reserved names
func HasNameCollisions(configs []OneTerminalConfig) bool {
	reservedNames := map[string]bool{
		"completion": true,
		"example":    true,
		"help":       true,
	}

	allNames := make(map[string]bool)
	for _, config := range configs {
		if allNames[config.Name] {
			fmt.Printf("duplicate command name used %s\n", config.Name)
			return true
		}
		allNames[config.Name] = true

		if reservedNames[config.Name] {
			fmt.Printf("reserved name used %s\n", config.Name)
			return true
		}
	}

	return false
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
				Command:     "echo hello $PERSON",
				Silence:     false,
				ReadyRegexp: "window [0-9]",
				DependsOn:   []string{"greeter-1"},
				Environment: map[string]string{"PERSON": "alex"},
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
#   5. depends-on {[]string, optional}: which (names of) commands to wait for
#   6. ready-regexp {string, optional}: a regular expression that the outputs
#        must match for this command to be considered "ready" and for its
#        dependants to begin running
#   7. environment {map[string]string, optional} to set environment variables
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
