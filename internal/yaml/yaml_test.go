package yaml

import (
	"os"
	"path"
	"testing"
)

// helper function that sets the configDir variable to the default temp directory
func setupTempDir(t *testing.T) {
	configDir = os.TempDir()
}

// testing utility function
func getFileContents(t *testing.T, path string) string {
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Reading file: %s; %v", path, err)
	}
	return string(bytes)
}

func TestMakeExampleConfigFromStruct(t *testing.T) {
	setupTempDir(t)

	filename := "oneterminal-example-from-struct.yml"
	if err := MakeExampleConfigFromStruct(filename); err != nil {
		t.Errorf("Expected no error from MakeExampleConfigFromStruct, got %s", err)
	}

	filepath := path.Join(configDir, filename)
	t.Logf("Wrote to temp file %s\n", filepath)
	defer os.Remove(filepath)

	fileContents := getFileContents(t, filepath)
	if len(fileContents) < 50 {
		t.Errorf("Expected file to be at least 50 characters, got %d", len(fileContents))
	}
}

func TestMakeExampleConfigWithInstructions(t *testing.T) {
	setupTempDir(t)

	filename := "oneterminal-example-with-instructions.yml"
	err := MakeExampleConfigWithInstructions(filename)
	if err != nil {
		t.Errorf("Did not expect error, got error %s", err)
	}

	filepath := path.Join(configDir, filename)
	t.Logf("Wrote to temp file %s\n", filepath)
	defer os.Remove(filepath)

	fileContents := getFileContents(t, filepath)

	if len(fileContents) < 50 {
		t.Errorf("Expected file to be at least 50 characters, got %d", len(fileContents))
	}
}

func TestParseAllConfigs(t *testing.T) {
	setupTempDir(t)

	// add an example config
	filename := "oneterminal-example-parse-configs.yml"
	MakeExampleConfigFromStruct(filename)
	t.Logf("Wrote to temp file %s\n", path.Join(configDir, filename))
	defer os.Remove(path.Join(configDir, filename))

	configs, err := ParseAllConfigs()
	if err != nil {
		t.Log("Relies on MakeExampleConfigFromStruct, ensure it is working.")
		t.Errorf("Did not expect error, got %s", err)
	}
	if len(configs) < 1 {
		t.Log("Relies on MakeExampleConfigFromStruct, ensure it is working.")
		t.Errorf("Expected at least one config, got %d", len(configs))
	}

	for _, config := range configs {
		// only check the generated example command
		if config.Name == "example-name" {
			if config.Shell != "zsh" {
				t.Errorf(`Expected shell to be set to "zsh", got %q"`, config.Shell)
			}
			if config.Alias != "exname" {
				t.Errorf(`want Alias %q, got %q"`, "exname", config.Alias)
			}
			if config.Short == "" {
				t.Errorf("Expected config's Short to not be an empty string, got %q", config.Short)
			}
			if config.Long == "" {
				t.Errorf("Expected config's Long to not be an empty string, got %q", config.Long)
			}
			if dir := config.Commands[0].CmdDir; dir != "$HOME/go" {
				t.Errorf("Expected config's first command to have CmdDir to be set to $HOME/go, got %q", dir)
			}
			if len(config.Commands) != 3 {
				t.Errorf("Expected there to be 3 commands, got %d", len(config.Commands))
			}
		}
	}
}
