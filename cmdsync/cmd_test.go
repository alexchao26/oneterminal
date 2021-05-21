package cmdsync

import (
	"errors"
	"log"
	"os/exec"
	"strings"
	"testing"
)

func getInstalledShells(t *testing.T) []string {
	var shells []string
	// find a shell that can be used for tests
	for _, s := range []string{"zsh", "bash", "sh"} {
		_, err := exec.LookPath(s)
		if err != nil {
			log.Printf("%q shell not installed", s)
		}
		shells = append(shells, s)
	}

	if len(shells) == 0 {
		t.Fatalf("no shells supported, tried zsh, bash and sh")
	}

	return shells
}

func TestShellCmd_Run(t *testing.T) {
	shells := getInstalledShells(t)

	tests := []struct {
		name                string
		command             string
		commandOpts         []ShellCmdOption
		wantOutput          string
		wantOutputToContain []string
		wantError           error
	}{
		{
			name:                "echo hello world",
			command:             "echo Hello, world!",
			commandOpts:         nil,
			wantOutput:          "Hello, world!\n",
			wantOutputToContain: []string{"Hello, world!"},
			wantError:           nil,
		},
		{
			name:                "go version",
			command:             "go version",
			commandOpts:         nil,
			wantOutputToContain: []string{"go version"},
			wantError:           nil,
		},
		{
			name:    "SetEnvironment Option",
			command: "echo $TEST_ENV_VAR",
			commandOpts: []ShellCmdOption{
				Environment(map[string]string{
					"TEST_ENV_VAR": "beepboop",
				}),
			},
			wantOutput: "beepboop\n",
			wantError:  nil,
		},
		{
			name:    "name prefixes output line",
			command: "echo potato",
			commandOpts: []ShellCmdOption{
				Name("cmdname"),
			},
			wantOutput: "cmdname | potato\n",
			wantError:  nil,
		},
		{
			name:      "command with non-zero exit code errors",
			command:   "exit 1",
			wantError: errors.New("exit status 1"),
		},
		{
			name:    "echo cmd then exit cmd",
			command: "echo hello && exit 1",
			commandOpts: []ShellCmdOption{
				Name("NAME"),
			},
			wantOutput: "NAME | hello\n",
			wantError:  errors.New("exit status 1"),
		},
	}

	// test all installed and supported shells
	for _, shell := range shells {
		t.Logf("using %s shell", shell)
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				shCmd, err := NewShellCmd(shell, tt.command, tt.commandOpts...)
				if err != nil {
					t.Errorf("NewShellCmd() error want nil, got %v", err)
				}

				var sb strings.Builder // implements io.Writer
				shCmd.stdout = &sb

				err = shCmd.Run()
				// also check error strings in case of non-sentinel errors
				if err != tt.wantError && err.Error() != tt.wantError.Error() {
					t.Errorf("shCmd.Run() want err %v, got %v", tt.wantError, err)
				}

				output := sb.String()
				if tt.wantOutput != "" && tt.wantOutput != output {
					t.Errorf("shCmd.Run() want %q, got %q", tt.wantOutput, output)
				}

				// check pieces because `go version` might output different ...versions
				for _, wantPiece := range tt.wantOutputToContain {
					if !strings.Contains(output, wantPiece) {
						t.Errorf("shCmd.Run() want output to contain %q, got %q", wantPiece, output)
					}
				}
			})
		}
	}
}

func TestPrefixEachLine(t *testing.T) {
	var tests = []struct {
		input, prefix, want string
	}{
		{"hi", "pre-1", "pre-1 | hi\n"},
		{"hello\nasdf", "shCmd", "shCmd | hello\nshCmd | asdf\n"},
		{"Starting...\nWaiting...\nReady!", "launcher", "launcher | Starting...\nlauncher | Waiting...\nlauncher | Ready!\n"},
	}

	for _, tt := range tests {
		actual := prefixEveryline(tt.input, tt.prefix)
		if actual != tt.want {
			t.Errorf("Expected %q, got %q", tt.want, actual)
		}
	}
}
