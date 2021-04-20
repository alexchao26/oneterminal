package cmdsync

import (
	"log"
	"os/exec"
	"strings"
	"testing"
)

var shell string

func init() {
	// find a shell that can be used for tests
	for _, s := range []string{"zsh", "bash", "sh"} {
		_, err := exec.LookPath(s)
		if err != nil {
			log.Printf("%q shell not installed", s)
			continue
		}
		shell = s
		return
	}
	panic("no supported shell installed, tried zsh, bash and sh")
}

func TestShellCmd_Run(t *testing.T) {
	tests := map[string]struct {
		command             string
		commandOpts         []ShellCmdOption
		wantOutputToContain []string
		wantErr             bool
	}{
		"echo Hello World": {"echo Hello, world!", nil, []string{"Hello, world!"}, false},
		"go version":       {"go version", nil, []string{"go version"}, false},
		"SetEnvironment Option": {
			"echo $TEST_ENV_VAR",
			[]ShellCmdOption{
				Environment(map[string]string{
					"TEST_ENV_VAR": "beepboop",
				}),
			}, []string{"beepboop"},
			false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Log(name, tt.command)
			shCmd, err := NewShellCmd(shell, tt.command, tt.commandOpts...)
			if err != nil {
				t.Errorf("NewShellCmd() error want nil, got %v", err)
			}

			var sb strings.Builder // implements io.Writer
			shCmd.command.Stdout = &sb
			shCmd.command.Stderr = &sb
			err = shCmd.Run()

			if tt.wantErr && err == nil {
				t.Errorf("shCmd.Run() want err, got nil")
			}
			if tt.wantErr == false && err != nil {
				t.Errorf("shCmd.Run() want err = nil, got %v", err)
			}

			outputs := sb.String()
			for _, wantPiece := range tt.wantOutputToContain {
				if !strings.Contains(outputs, wantPiece) {
					t.Errorf("shCmd.Run() = %q, want it to contain %q", outputs, wantPiece)
				}
			}
		})
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
