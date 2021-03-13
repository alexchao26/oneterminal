package cmdsync

import (
	"strings"
	"testing"
)

func TestMonitoredCmd_Run(t *testing.T) {
	for name, tc := range map[string]struct {
		command             string
		commandOpts         []CmdOption
		wantOutputToContain []string
	}{
		"echo Hello World": {"echo Hello, world!", nil, []string{"Hello, world!"}},
		"go version":       {"go version", nil, []string{"go version"}},
		"SetEnvironment Option": {
			"echo $TEST_ENV_VAR",
			[]CmdOption{
				SetEnvironment(map[string]string{
					"TEST_ENV_VAR": "beepboop",
				}),
			}, []string{"beepboop"}},
	} {
		closure := tc
		t.Run(name, func(tt *testing.T) {
			cmd := NewCmd(closure.command, closure.commandOpts...)

			var sb strings.Builder // implements io.Writer
			cmd.command.Stdout = &sb
			cmd.command.Stderr = &sb
			cmd.Run()

			outputs := sb.String()

			for _, wantPiece := range closure.wantOutputToContain {
				if !strings.Contains(outputs, wantPiece) {
					tt.Errorf("cmd.Run() = %s, want it to contain %s", outputs, wantPiece)
				}
			}
		})
	}
}

var prefixEachLineTests = []struct {
	input, prefix, want string
}{
	{"hi", "pre-1", "pre-1 | hi\n"},
	{"hello\nasdf", "cmd", "cmd | hello\ncmd | asdf\n"},
	{"Starting...\nWaiting...\nReady!", "launcher", "launcher | Starting...\nlauncher | Waiting...\nlauncher | Ready!\n"},
}

func TestPrefixEachLine(t *testing.T) {
	for _, test := range prefixEachLineTests {
		actual := prefixEveryline(test.input, test.prefix)
		if actual != test.want {
			t.Errorf("Expected %q, got %q", test.want, actual)
		}
	}
}
