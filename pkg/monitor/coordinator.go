package monitor

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Coordinator uses a channel for commands to communicate their donness
// and has a mutex to prevent races against a pendingCmdCount
type Coordinator struct {
	commands []MonitoredCmd
	wg       sync.WaitGroup
}

// NewCoordinator makes a new coordinator
// it can be optionally initialized with commands
// or they can be added later via AddCommands
func NewCoordinator(commands ...MonitoredCmd) *Coordinator {
	return &Coordinator{
		commands: append([]MonitoredCmd{}, commands...),
	}
}

// AddCommands will add MonitoredCmds to the commands slice
// increment pendingCmdCount
func (coord *Coordinator) AddCommands(commands ...MonitoredCmd) {
	// does not require a mutex/lock because the API is designed
	// to have all commands added prior to running, i.e. synchronously
	coord.commands = append(coord.commands, commands...)
}

// RunCommands will run all of the added commands and block until they have all
// finished running. The can occur from the processes ending naturally
// or being interrupted
// TODO - timing of when to run each command?
func (coord *Coordinator) RunCommands() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

	go func() {
		<-signalChan
		coord.SendInterrupt()
	}()

	for _, cmd := range coord.commands {
		coord.wg.Add(1)
		go func(cmd MonitoredCmd) {
			err := cmd.Run()
			if err != nil {
				fmt.Println(err)
			}
			coord.wg.Done()
		}(cmd)
	}

	coord.wg.Wait()
}

// SendInterrupt will relay an interrupt signal to all underlying commands
func (coord *Coordinator) SendInterrupt() {
	for _, cmd := range coord.commands {
		// TODO add a check for the command's status, if it is already done
		// TODO then don't send an interrupt signal
		cmd.Interrupt()
	}
}
