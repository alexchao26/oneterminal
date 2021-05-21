package cmdsync

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Group manages scheduling concurrent ShellCmds
type Group struct {
	commands   []*ShellCmd
	hasStarted bool
	mut        sync.RWMutex
}

// NewGroup makes a new Group
// it can be optionally initialized with commands
// or they can be added later via AddCommands
func NewGroup(commands ...*ShellCmd) *Group {
	return &Group{
		commands: commands,
	}
}

// AddCommands will add ShellCmds to the commands slice
// It will return an error if called after Group.Run()
func (g *Group) AddCommands(commands ...*ShellCmd) error {
	g.mut.Lock()
	defer g.mut.Unlock()
	if g.hasStarted {
		return fmt.Errorf("Group has already been started")
	}
	g.commands = append(g.commands, commands...)
	return nil
}

// Run will run all of the group's ShellCmds and block until they have all
// finished running or an interrupt signal is sent (ctrl + c). Internally it
// relays the first interrupt signal to all underlying ShellCmds. Additional
// interrupt commands will return to normal behavior.
//
// It checks for each ShellCmd's prerequisites before starting. See ShellCmd for
// details on ready regexp.
//
// The returned error is the first error returned from any of the Group's
// ShellCmds, if any.
func (g *Group) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	go func() {
		<-ctx.Done()
		stop()
	}()
	return g.RunContext(ctx)
}

// RunContext is the same as Run but does not setup singal notifying internally.
// This means callers can only interrupt the Group's ShellCmds by cancelling the
// context.
//
// To cancel the context via an interrupt signal from the terminal (ctrl + c),
// use signal.NotifyContext.
//   ctx, done := signal.NotifyContext(context.Background(), os.Interrupt)
//   // ensure done() is called to restore normal SIGINT behavior
//   go func() {
//       <- ctx.Done()
//       done()
//   }()
//   err := group.Run(ctx)
//   // handle error
func (g *Group) RunContext(ctx context.Context) error {
	g.mut.Lock()
	g.hasStarted = true
	g.mut.Unlock()

	namesToCmds := make(map[string]*ShellCmd, len(g.commands))
	for _, cmd := range g.commands {
		namesToCmds[cmd.name] = cmd
	}

	eg, ctx := errgroup.WithContext(ctx)

	go func() {
		<-ctx.Done()
		g.SendInterrupts()
	}()

	for _, cmd := range g.commands {
		// https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		cmd := cmd
		eg.Go(func() error {
			ticker := time.NewTicker(time.Millisecond * 200)
			defer ticker.Stop()
			// on every tick, exit if context is done (shutdown has started)
			// then start command if all depends-on ShellCmds' are in a ready state
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
				}

				canStart, err := checkDependencies(cmd, namesToCmds)
				if err != nil {
					return fmt.Errorf("%s: %w", cmd.name, err)
				}
				if canStart {
					ticker.Stop()
					err := cmd.Run()
					if err != nil {
						return fmt.Errorf("%s: %w", cmd.name, err)
					}
					return nil
				}
			}
		})
	}

	return eg.Wait()
}

// SendInterrupts relays an interrupt signal to all underlying commands
func (g *Group) SendInterrupts() {
	if !g.hasStarted {
		return
	}
	for _, cmd := range g.commands {
		cmd.Interrupt()
	}
}

func checkDependencies(cmd *ShellCmd, allCmdsMap map[string]*ShellCmd) (bool, error) {
	for _, depName := range cmd.dependsOn {
		depCmd, ok := allCmdsMap[depName]
		if !ok {
			return false, fmt.Errorf("%q depends-on %q, but %q does not exist", cmd.name, depName, depName)
		}
		if cmd.name == depName {
			return false, fmt.Errorf("%s depends on itself", cmd.name)
		}
		if !depCmd.IsReady() {
			return false, nil
		}
	}
	return true, nil
}
