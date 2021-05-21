package cmdsync

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestGroup_RunContext(t *testing.T) {
	testShell := getInstalledShells(t)[0]
	t.Logf("using %s shell", testShell)

	mustNewShellCmd := func(testShell, command string, opts ...ShellCmdOption) *ShellCmd {
		cmd, err := NewShellCmd(testShell, command, opts...)
		if err != nil {
			t.Fatalf("malformed test, failed mustNewShellCmd(%s, %s, opts...), err: %s", testShell, command, err)
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
				mustNewShellCmd(testShell, "echo monkeypotato", Name("first")),
				mustNewShellCmd(testShell, "echo next", Name("second"), DependsOn("first")),
				mustNewShellCmd(testShell, "echo last", Name("last"), DependsOn("second")),
			),
			wantOutput: "first | monkeypotato\nsecond | next\nlast | last\n",
			wantError:  nil,
		},
		{
			name: "different depends on ordering",
			group: NewGroup(
				mustNewShellCmd(testShell, "echo monkeypotato", Name("first"), DependsOn("second")),
				mustNewShellCmd(testShell, "echo next", Name("second")),
				mustNewShellCmd(testShell, "echo last", Name("last"), DependsOn("second", "first")),
			),
			wantOutput: "second | next\nfirst | monkeypotato\nlast | last\n",
			wantError:  nil,
		},
		{
			name: "silent middle command does not print to stdout",
			group: NewGroup(
				mustNewShellCmd(testShell, "echo monkeypotato", Name("first")),
				mustNewShellCmd(testShell, "echo next", Name("second"), DependsOn("first"), SilenceOutput()),
				mustNewShellCmd(testShell, "echo last", Name("last"), DependsOn("second", "first")),
			),
			wantOutput: "first | monkeypotato\nlast | last\n",
			wantError:  nil,
		},
		{
			name: "ready regexp allows dependent commands to start concurrently",
			group: NewGroup(
				mustNewShellCmd(testShell, "echo next", Name("second"), DependsOn("first")),
				mustNewShellCmd(testShell, "echo last", Name("last"), DependsOn("second", "first")),
				mustNewShellCmd(testShell, "echo monkeypotato && sleep 1 && echo finally",
					Name("first"),
					ReadyPattern("monkey"),
				),
			),
			wantOutput: "first | monkeypotato\nsecond | next\nlast | last\nfirst | finally\n",
			wantError:  nil,
		},
		{
			name: "test with echo, cat and rm commands",
			group: NewGroup(
				mustNewShellCmd(testShell, "echo 'file contents' > asdf.txt",
					Name("write"),
					CmdDir(os.TempDir()),
				),
				mustNewShellCmd(testShell, "cat asdf.txt",
					Name("read"),
					CmdDir(os.TempDir()),
					DependsOn("write"),
				),
				mustNewShellCmd(testShell, "rm asdf.txt",
					Name("remove"),
					CmdDir(os.TempDir()),
					DependsOn("read", "write"),
				),
			),
			wantOutput: "read | file contents\n",
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
				mustNewShellCmd(testShell, "echo monkeypotato", Name("first")),
				mustNewShellCmd(testShell, "echo next", Name("second"), DependsOn("first"), SilenceOutput()),
				mustNewShellCmd(testShell, "echo last", Name("last"), DependsOn("second", "first")),
			),
			wantOutput: "",
			wantError:  context.Canceled,
		},
		{
			name: "a command exits with non-zero code",
			group: NewGroup(
				mustNewShellCmd(testShell, "exit 1", Name("unhappy cmd")),
			),
			wantOutput: "",
			wantError:  errors.New("unhappy cmd: exit status 1"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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

			if err != tt.wantError {
				// also check error strings in case of non-sentinel errors
				want, got := "nil", "nil"
				if tt.wantError != nil {
					want = tt.wantError.Error()
				}
				if err != nil {
					got = err.Error()
				}
				if want != got {
					t.Errorf("shCmd.Run() want err %q, got %q", want, got)
				}
			}

			gotOutput := sb.String()
			if gotOutput != tt.wantOutput {
				t.Errorf("group stdout, want %q, got %q", tt.wantOutput, gotOutput)
			}
		})
	}
}
