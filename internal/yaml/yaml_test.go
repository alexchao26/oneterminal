package yaml

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

func TestGetConfigDir(t *testing.T) {
	dir, err := GetConfigDir()
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
	if !strings.HasSuffix(dir, "/.config/oneterminal") {
		t.Errorf("Expected directory to end in \"/.config/oneterminal\", path was %s", dir)
	}
}

// testing utility function
func getFileContents(path string) (string, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return "", errors.Wrapf(err, "error reading file %s", path)
	}

	return string(bytes), nil
}

func TestMakeExampleConfigFromStruct(t *testing.T) {
	if err := MakeExampleConfigFromStruct(); err != nil {
		t.Errorf("Expected no error from MakeExampleConfigFromStruct, got %s", err)
	}

	filepath := os.ExpandEnv("$HOME/.config/oneterminal/generated-example.yml")
	fileContents, err := getFileContents(filepath)
	if err != nil {
		t.Errorf("Error reading the generated config file %s", err)
	}
	if len(fileContents) < 50 {
		t.Errorf("Expected file to be at least 50 characters, got %d", len(fileContents))
	}

	// remove example file
	os.Remove(filepath)
}

func TestMakeExampleConfigFromStructWithInstructions(t *testing.T) {
	err := MakeExampleConfigFromStructWithInstructions()
	if err != nil {
		t.Errorf("Did not expect error, got error %s", err)
	}

	filepath := os.ExpandEnv("$HOME/.config/oneterminal/example.yml")
	fileContents, err := getFileContents(filepath)
	if err != nil {
		t.Errorf("Error reading the generated config file %s", err)
	}
	if len(fileContents) < 50 {
		t.Errorf("Expected file to be at least 50 characters, got %d", len(fileContents))
	}

	// remove example file
	os.Remove(filepath)
}

func TestParseConfigs(t *testing.T) {
	// add at least one config
	MakeExampleConfigFromStruct()
	configs, err := ParseConfigs()
	if err != nil {
		t.Errorf("Did not expect error, got %s", err)
	}
	if len(configs) < 1 {
		t.Errorf("Expected at least one config, got %d", len(configs))
	}

	for _, config := range configs {
		// find the generated command
		if config.Name == "somename" {
			if config.Shell != "zsh" {
				t.Errorf("Expected shell to be set to \"zsh\", got %q", config.Shell)
			}
			if config.Short == "" {
				t.Errorf("Expected config's Short to not be an empty string, got %q", config.Short)
			}
			if config.Long == "" {
				t.Errorf("Expected config's Long to not be an empty string, got %q", config.Long)
			}
			if config.CmdDir != "$HOME/go" {
				t.Errorf("Expected config's CmdDir to be set to $HOME/go, got %q", config.CmdDir)
			}
			if len(config.Commands) != 3 {
				t.Errorf("Expected there to be 3 commands, got %d", len(config.Commands))
			}
		}
	}

	// delete that file when done
	filepath := os.ExpandEnv("$HOME/.config/oneterminal/generated-example.yml")
	os.Remove(filepath)
}
