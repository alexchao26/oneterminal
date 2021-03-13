package cmdsync

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// Group manages scheduling concurrent Cmds
type Group struct {
	commands      []*Cmd
	hasStarted    bool
	isInterrupted bool
	mut           sync.RWMutex
}

// NewGroup makes a new Group
// it can be optionally initialized with commands
// or they can be added later via AddCommands
func NewGroup(commands ...*Cmd) *Group {
	return &Group{
		commands: commands,
	}
}

// AddCommands will add Cmds to the commands slice
// It will return an error if called after Group.Run()
func (g *Group) AddCommands(commands ...*Cmd) error {
	g.mut.Lock()
	defer g.mut.Unlock()
	if g.hasStarted {
		return errors.New("Group has already been started")
	}
	g.commands = append(g.commands, commands...)
	return nil
}

// Run will run all of the group's Cmds and block until they have all finished
// running, or an interrupt/kill signal is received, or the context cancels
//
// It checks for each Cmd's prerequisites (Cmds it depends-on being in a ready
// state) before starting the Cmd
//
// The returned error is the first error returned from the Group's Cmds, if any
func (g *Group) Run(ctx context.Context) error {
	g.hasStarted = true

	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, os.Interrupt, os.Kill)
	go func() {
		select {
		case <-ctx.Done():
		case <-signalChan:
		}
		g.mut.Lock()
		g.isInterrupted = true
		g.mut.Unlock()
		g.SendInterrupts()
	}()
	defer close(signalChan)

	namesToCmds := make(map[string]*Cmd, len(g.commands))
	for _, cmd := range g.commands {
		namesToCmds[cmd.name] = cmd
	}

	eg, ctx := errgroup.WithContext(ctx)

	for _, cmd := range g.commands {
		cmd := cmd
		eg.Go(func() error {
			ticker := time.NewTicker(time.Millisecond * 200)
			defer ticker.Stop()
			// on every tick, exit if context has been cancelled or group's been
			// interrupted. then check all depends-on Cmds' ready state
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
				}

				g.mut.RLock()
				if g.isInterrupted {
					g.mut.RUnlock()
					return nil
				}
				g.mut.RUnlock()

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

func checkDependencies(m *Cmd, allCmdsMap map[string]*Cmd) (bool, error) {
	for _, depName := range m.dependsOn {
		depCmd, ok := allCmdsMap[depName]
		if !ok {
			return false, errors.Errorf("%q depends-on %q, but %q does not exist", m.name, depName, depName)
		}
		if m.name == depName {
			return false, errors.Errorf("%s depends on itself", m.name)
		}
		if !depCmd.IsReady() {
			return false, nil
		}
	}
	return true, nil
}
