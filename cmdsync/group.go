package cmdsync

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
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
		return errors.New("Group has already been started")
	}
	g.commands = append(g.commands, commands...)
	return nil
}

// Run will run all of the group's ShellCmds and block until they have all finished
// running, or an interrupt/kill signal is received
//
// It checks for each ShellCmd's prerequisites (ShellCmds it depends-on being in a ready
// state) before starting the ShellCmd
//
// The returned error is the first error returned from the Group's ShellCmds, if any
func (g *Group) Run() error {
	return g.RunContext(context.Background())
}

// RunContext is the same as Run but can also be cancelled via ctx
func (g *Group) RunContext(ctx context.Context) error {
	g.mut.Lock()
	g.hasStarted = true
	g.mut.Unlock()

	namesToCmds := make(map[string]*ShellCmd, len(g.commands))
	for _, cmd := range g.commands {
		namesToCmds[cmd.name] = cmd
	}

	eg, ctx := errgroup.WithContext(ctx)

	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		// events that could lead to a shutdown:
		// 1. ctx is ended (from parent or errgroup Go routine returning non-nil error)
		// 2. signal is received (ctrl + c)
		select {
		case <-ctx.Done():
		case <-signalChan:
		}
		g.SendInterrupts()
	}()
	defer close(signalChan)

	for _, cmd := range g.commands {
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
					return errors.Wrap(err, cmd.name)
				}
				if canStart {
					ticker.Stop()
					err := cmd.Run()
					if err != nil {
						return errors.Wrap(err, cmd.name)
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
			return false, errors.Errorf("%q depends-on %q, but %q does not exist", cmd.name, depName, depName)
		}
		if cmd.name == depName {
			return false, errors.Errorf("%s depends on itself", cmd.name)
		}
		if !depCmd.IsReady() {
			return false, nil
		}
	}
	return true, nil
}
