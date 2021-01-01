package monitor

import (
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
// finished running. The can occur from the processes ending naturally
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
		orch.wg.Add(1)
		go func(cmd *MonitoredCmd) {
			ticker := time.NewTicker(time.Millisecond * 100)
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

				if checkDependencies(cmd, namesToCmds) {
					err := cmd.Run()
					if err != nil {
						fmt.Println(err)
					}
					break
				}
			}
			orch.wg.Done()
		}(cmd)
	}

	orch.wg.Wait()
}

// SendInterrupts will relay an interrupt signal to all underlying commands
func (orch *Orchestrator) SendInterrupts() {
	for _, cmd := range orch.commands {
		cmd.Interrupt()
	}
}

func checkDependencies(m *MonitoredCmd, allCmdsMap map[string]*MonitoredCmd) bool {
	for _, dep := range m.dependsOn {
		if !allCmdsMap[dep].IsReady() {
			return false
		}
	}
	return true
}
