// Package yaml is an internal package for parsing config files into a format
// that is similar to cobra commands.
package yaml

import (
	_ "embed"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// intialized by init(), can be changed for running tests in a tmp directory
var configDir string

//go:embed example.yaml
var exampleConfig []byte

func init() {
	// initialize configDir, this allows it to be changed for tests
	homedir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("getting home directory %v", err))
	}
	configDir = filepath.Join(homedir, ".config/oneterminal")
	if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
		panic(fmt.Sprintf("making config dir ~/.config/oneterminal %v", err))
	}
}

// OneTerminalConfig of all the fields from a yaml config
type OneTerminalConfig struct {
	Name     string    `yaml:"name"`
	Alias    string    `yaml:"alias"`
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

var isYamlPattern = regexp.MustCompile(".ya?ml$")

// ParseAllConfigs parses and returns configs in ~/.config/oneterminal
func ParseAllConfigs() ([]OneTerminalConfig, error) {
	// Unmarshal all values from configDir
	var allConfigs []OneTerminalConfig
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, errors.Wrap(err, "reading from config directory")
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !isYamlPattern.MatchString(e.Name()) {
			continue
		}

		filename := path.Join(configDir, e.Name())

		bytes, err := os.ReadFile(filename)
		if err != nil {
			return nil, errors.Wrapf(err, "reading file %s", filename)
		}
		var oneTermConfig OneTerminalConfig
		err = yaml.Unmarshal(bytes, &oneTermConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "unmarshalling file %s", filename)
		}
		err = validateConfig(oneTermConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid config from %q", filename)
		}

		allConfigs = append(allConfigs, oneTermConfig)
	}

	return allConfigs, nil
}

// non-exhaustive validation, checks for required fields
func validateConfig(config OneTerminalConfig) error {
	if config.Name == "" {
		return fmt.Errorf("missing name")
	}
	if regexp.MustCompile("`^(zsh|sh|bash)$`").MatchString(config.Shell) {
		return fmt.Errorf("%s shell not supported", config.Shell)
	}
	if len(config.Commands) == 0 {
		return fmt.Errorf("no commands configured")
	}

	for i, cmd := range config.Commands {
		if cmd.Command == "" {
			return fmt.Errorf("cmd no. %d is missing command field", i)
		}
	}

	return nil
}

// HasNameCollisions returns an error if multiple configs have the same name,
// alias or one of the reserved names (for built in oneterminal cmds like help)
func HasNameCollisions(configs []OneTerminalConfig) error {
	reservedNames := map[string]bool{
		"completion": true,
		"example":    true,
		"help":       true,
		"list":       true,
		"ls":         true,
		"update":     true,
	}

	allNames := make(map[string]bool)
	for _, config := range configs {
		if allNames[config.Name] || allNames[config.Alias] {
			return errors.Errorf("duplicate name or alias used: %q or %q", config.Name, config.Alias)
		}

		if reservedNames[config.Name] || reservedNames[config.Alias] {
			return errors.Errorf("reserved name used: %q or %q", config.Name, config.Alias)
		}

		allNames[config.Name] = true
		if config.Alias != "" {
			allNames[config.Alias] = true
		}
	}

	return nil
}

// WriteExampleConfig makes an example oneterminal yaml config at
// ~/.config/oneterminal/example.yml with comments describing each field
func WriteExampleConfig(filename string) error {
	err := os.WriteFile(path.Join(configDir, filename), exampleConfig, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "writing to example config file")
	}

	return nil
}
