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
	commands []*MonitoredCmd
	wg       sync.WaitGroup
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
	// does not require a mutex/lock because the API is designed
	// to have all commands added prior to running, i.e. synchronously
	orch.commands = append(orch.commands, commands...)
}

// RunCommands will run all of the added commands and block until they have all
// finished running. The can occur from the processes ending naturally
// or being interrupted
func (orch *Orchestrator) RunCommands() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

	go func() {
		<-signalChan
		orch.SendInterrupt()
	}()

	namesToCmds := make(map[string]*MonitoredCmd)
	for _, cmd := range orch.commands {
		namesToCmds[cmd.name] = cmd
	}

	for _, cmd := range orch.commands {
		orch.wg.Add(1)
		go func(cmd *MonitoredCmd) {
			ticker := time.NewTicker(time.Millisecond * 100)
			for {
				<-ticker.C
				if checkDependencies(cmd, namesToCmds) {
					err := cmd.Run()
					if err != nil {
						panic(fmt.Sprintf("Error running command %s: %s", cmd.name, err))
					}
					break
				}
			}

			orch.wg.Done()
		}(cmd)
	}

	orch.wg.Wait()
}

// SendInterrupt will relay an interrupt signal to all underlying commands
func (orch *Orchestrator) SendInterrupt() {
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
