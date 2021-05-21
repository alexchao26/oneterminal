package cmdsync

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestGroup_RunContext(t *testing.T) {
	shell := getInstalledShells(t)[0]
	t.Logf("using %s shell", shell)

	mustNewShellCmd := func(shell, command string, opts ...ShellCmdOption) *ShellCmd {
		cmd, err := NewShellCmd(shell, command, opts...)
		if err != nil {
			t.Fatalf("malformed test, failed mustNewShellCmd(%s, %s, opts...), err: %s", shell, command, err)
		}
		return cmd
	}

	tests := []struct {
		name       string
		ctx        context.Context
		group      *Group
		wantOutput string
		wantError  error
	}{
		{
			name: "commands occur in order",
			group: NewGroup(
				mustNewShellCmd(shell, "echo monkeypotato", Name("first")),
				mustNewShellCmd(shell, "echo next", Name("second"), DependsOn("first")),
				mustNewShellCmd(shell, "echo last", Name("last"), DependsOn("second")),
			),
			wantOutput: "first | monkeypotato\nsecond | next\nlast | last\n",
			wantError:  nil,
		},
		{
			name: "different depends on ordering",
			group: NewGroup(
				mustNewShellCmd(shell, "echo monkeypotato", Name("first"), DependsOn("second")),
				mustNewShellCmd(shell, "echo next", Name("second")),
				mustNewShellCmd(shell, "echo last", Name("last"), DependsOn("second", "first")),
			),
			wantOutput: "second | next\nfirst | monkeypotato\nlast | last\n",
			wantError:  nil,
		},
		{
			name: "silent middle command does not print to stdout",
			group: NewGroup(
				mustNewShellCmd(shell, "echo monkeypotato", Name("first")),
				mustNewShellCmd(shell, "echo next", Name("second"), DependsOn("first"), SilenceOutput()),
				mustNewShellCmd(shell, "echo last", Name("last"), DependsOn("second", "first")),
			),
			wantOutput: "first | monkeypotato\nlast | last\n",
			wantError:  nil,
		},
		{
			name: "ready regexp is followed",
			group: NewGroup(
				mustNewShellCmd(shell, "echo next", Name("second"), DependsOn("first")),
				mustNewShellCmd(shell, "echo last", Name("last"), DependsOn("second", "first")),
				mustNewShellCmd(shell, "echo monkeypotato && sleep 0.5", Name("first"), ReadyPattern("monkey")),
			),
			wantOutput: "first | monkeypotato\nsecond | next\nlast | last\n",
			wantError:  nil,
		},
		{
			name: "already cancelled context errors with context.Canceled",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // cancel it
				return ctx
			}(),
			group: NewGroup(
				mustNewShellCmd(shell, "echo monkeypotato", Name("first")),
				mustNewShellCmd(shell, "echo next", Name("second"), DependsOn("first"), SilenceOutput()),
				mustNewShellCmd(shell, "echo last", Name("last"), DependsOn("second", "first")),
			),
			wantOutput: "",
			wantError:  context.Canceled,
		},
		{
			name: "a command exits with non-zero code",
			group: NewGroup(
				mustNewShellCmd(shell, "exit 1", Name("unhappy cmd")),
			),
			wantOutput: "",
			wantError:  errors.New("unhappy cmd: exit status 1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// default context
			if tt.ctx == nil {
				tt.ctx = context.Background()
			}

			// modify internal stdouts for testability
			var sb strings.Builder
			for i := range tt.group.commands {
				tt.group.commands[i].stdout = &sb
			}

			err := tt.group.RunContext(tt.ctx)

			// also attempt to match error strings in case if it's not a sentinel error
			if err != tt.wantError && err.Error() != tt.wantError.Error() {
				t.Errorf("group.RunContext() want error %v, got %v", tt.wantError, err)
			}

			gotOutput := sb.String()
			if gotOutput != tt.wantOutput {
				t.Errorf("group stdout, want %q, got %q", tt.wantOutput, gotOutput)
			}
		})
	}
}
