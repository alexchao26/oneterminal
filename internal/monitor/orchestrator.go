package monitor

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Orchestrator uses a channel for commands to communicate their donness
// and has a mutex to prevent races against a pendingCmdCount
type Orchestrator struct {
	commands      []*MonitoredCmd
	isInterrupted bool
	mut           sync.RWMutex
	wg            sync.WaitGroup
}

// NewOrchestrator makes a new Orchestrator
// it can be optionally initialized with commands
// or they can be added later via AddCommands
func NewOrchestrator(commands ...*MonitoredCmd) *Orchestrator {
	return &Orchestrator{
		commands: append([]*MonitoredCmd{}, commands...),
	}
}

// AddCommands will add MonitoredCmds to the commands slice
// increment pendingCmdCount
func (orch *Orchestrator) AddCommands(commands ...*MonitoredCmd) {
	orch.mut.Lock() // not really necessary
	orch.commands = append(orch.commands, commands...)
	orch.mut.Unlock()
}

// RunCommands will run all of the added commands and block until they have all
// finished running. This can occur from the processes ending naturally
// or being interrupted
func (orch *Orchestrator) RunCommands() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

	go func() {
		<-signalChan
		orch.mut.Lock()
		orch.isInterrupted = true
		orch.mut.Unlock()
		orch.SendInterrupts()
	}()

	namesToCmds := make(map[string]*MonitoredCmd)
	for _, cmd := range orch.commands {
		namesToCmds[cmd.name] = cmd
	}

	for _, cmd := range orch.commands {
		cmd := cmd
		orch.wg.Add(1)
		go func() {
			defer orch.wg.Done()
			ticker := time.NewTicker(time.Millisecond * 200)
			defer ticker.Stop()
			// on every tick. check if entire orchestrator has been interrupted
			// then check dependencies of of this command, run it if unblocked
			for {
				<-ticker.C

				orch.mut.RLock()
				if orch.isInterrupted {
					orch.mut.RUnlock()
					break
				}
				orch.mut.RUnlock()

				canStart, err := checkDependencies(cmd, namesToCmds)
				if err != nil {
					fmt.Println(err)
					close(signalChan)
					return
				}
				if canStart {
					ticker.Stop() // safe to call twice?
					err := cmd.Run()
					if err != nil {
						// TODO close the signalChan to send interrupts to other processes b/c a failed dependency should interrupt all other dependencies
						// TODO add error messaging here if the err is from something other than an interrupt signal
						// fmt.Printf("Error running %s: %v\n", cmd.name, err)
						fmt.Println(err)
						close(signalChan)
					}
					break
				}
			}
		}()
	}

	orch.wg.Wait()
}

// SendInterrupts will relay an interrupt signal to all underlying commands
func (orch *Orchestrator) SendInterrupts() {
	for _, cmd := range orch.commands {
		cmd.Interrupt()
	}
}

func checkDependencies(m *MonitoredCmd, allCmdsMap map[string]*MonitoredCmd) (bool, error) {
	for _, depName := range m.dependsOn {
		depCmd, ok := allCmdsMap[depName]
		if !ok {
			return false, errors.New(fmt.Sprintf("%q depends-on %q, but %q does not exist", m.name, depName, depName))
		}
		if !depCmd.IsReady() {
			return false, nil
		}
	}
	return true, nil
}
