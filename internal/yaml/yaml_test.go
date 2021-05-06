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
		t.Fatalf("Reading file: %s: %v", path, err)
	}
	return string(bytes)
}

func TestMakeExampleConfigWithInstructions(t *testing.T) {
	setupTempDir(t)

	filename := "oneterminal-example-with-instructions.yml"
	err := WriteExampleConfig(filename)
	if err != nil {
		t.Errorf("want nil error, got %s", err)
	}

	filepath := path.Join(configDir, filename)
	t.Logf("Wrote to temp file %s\n", filepath)
	defer os.Remove(filepath)

	fileContents := getFileContents(t, filepath)

	if len(fileContents) < 50 {
		t.Errorf("want file to be at least 50 characters, got %d", len(fileContents))
	}
}

func TestParseAllConfigs(t *testing.T) {
	setupTempDir(t)

	// add an example config
	filename := "oneterminal-example-parse-configs.yml"
	WriteExampleConfig(filename)
	t.Logf("Wrote to temp file %s\n", path.Join(configDir, filename))
	defer os.Remove(path.Join(configDir, filename))

	configs, err := ParseAllConfigs()
	if err != nil {
		t.Log("Relies on MakeExampleConfigFromStruct, ensure it is working.")
		t.Errorf("want nil error, got %s", err)
	}
	if len(configs) < 1 {
		t.Log("Relies on MakeExampleConfigFromStruct, ensure it is working.")
		t.Errorf("want >= 1 config, got %d", len(configs))
	}

	for _, config := range configs {
		// only check the generated example command
		if config.Name == "example-name" {
			if config.Shell != "zsh" {
				t.Errorf(`want shell "zsh", got %q"`, config.Shell)
			}
			if config.Alias != "exname" {
				t.Errorf(`want Alias %q, got %q"`, "exname", config.Alias)
			}
			if config.Short == "" {
				t.Errorf("want config.Short to not be an empty string")
			}
			if config.Long == "" {
				t.Errorf("want config.Long to not be an empty string")
			}
			if dir := config.Commands[1].CmdDir; dir != "$HOME/go" {
				t.Errorf("want config.Commands[1].CmdDir to be \"$HOME/go\", got %q", dir)
			}
			if len(config.Commands) != 3 {
				t.Errorf("want command count to be 3, got %d", len(config.Commands))
			}
		}
	}
}
