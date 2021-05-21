package cmdsync_test

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/alexchao26/oneterminal/cmdsync"
)

func ExampleGroup_Run() {
	cmd1, _ := cmdsync.NewShellCmd("bash", "echo logging into vault && sleep 0.5 && echo logged in",
		cmdsync.Name("setup"),
		// a regexp pattern that must match the command's outputs for it to be deemed "ready", and
		// for its dependents to start executing
		cmdsync.ReadyPattern("logged in"),
	)
	cmd2, _ := cmdsync.NewShellCmd("bash", "echo starting some api...",
		cmdsync.Name("second"),
		// will not start until the "setup" command is "ready"
		cmdsync.DependsOn("setup"),
	)
	cmd3, _ := cmdsync.NewShellCmd("bash", "echo sweep sweep",
		cmdsync.Name("cleanup"),
		// will happen last
		cmdsync.DependsOn("second"),
	)

	group := cmdsync.NewGroup(cmd1, cmd2, cmd3)
	err := group.Run()
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// setup | logging into vault
	// setup | logged in
	// second | starting some api...
	// cleanup | sweep sweep
}

func ExampleGroup_RunContext_notify() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	// RunContext might hang even when interrupted, so reset the signal behavior
	// in a goroutine to ensure subsequent interrupt signals behave "normally".
	go func() {
		<-ctx.Done()
		stop()
	}()

	cmd1, _ := cmdsync.NewShellCmd("bash", "echo potatoes", cmdsync.Name("first"))
	cmd2, _ := cmdsync.NewShellCmd("bash", "echo are", cmdsync.Name("second"), cmdsync.DependsOn("first"))
	cmd3, _ := cmdsync.NewShellCmd("bash", "echo great", cmdsync.Name("third"), cmdsync.DependsOn("second"))

	group := cmdsync.NewGroup(cmd1, cmd2, cmd3)
	err := group.RunContext(ctx)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// first | potatoes
	// second | are
	// third | great
}
