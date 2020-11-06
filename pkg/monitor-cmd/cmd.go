package monitor

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
)

// MonitoredCmd is a wrapper around exec.Cmd
// its implementations calls the shell directly and
// sending the process a termination/interruption signal
// and checking if the process has completed
type MonitoredCmd struct {
	signalChan  chan syscall.Signal
	done        chan bool
	coordinator *Coordinator
	*exec.Cmd
}

// NewMonitoredCmd makes a command that can be interrupted
// via its signalChan channel, which is exposed by its
// Interrupt method
// Default shell used is zsh, use functional options to change
// e.g. monitor.NewMonitoredCmd("echo hello", monitor.BashShell)
func NewMonitoredCmd(command string, coordinator *Coordinator, options ...func(MonitoredCmd) MonitoredCmd) MonitoredCmd {
	c := exec.Command("zsh", "-c", command)

	c.Stdout = os.Stdout
	c.Stderr = os.Stdout

	// this sets the child process's PID to be the parent's PID
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	m := MonitoredCmd{
		Cmd:         c,
		signalChan:  make(chan syscall.Signal, 1),
		done:        make(chan bool, 1),
		coordinator: coordinator,
	}

	// increment coordinator's pending job count
	coordinator.PendingCmdCount++

	for _, f := range options {
		m = f(m)
	}

	return m
}

// BashShell is a functional option to change the executing shell to zsh
func BashShell(m MonitoredCmd) MonitoredCmd {
	m.Cmd.Args[0] = "bash"
	resolvedPath, err := exec.LookPath("bash")
	if err != nil {
		fmt.Println("Error setting bash as shell", err)
	}
	m.Cmd.Path = resolvedPath
	fmt.Println("cmd args are", m.Cmd.Args)
	return m
}

// SetCmdDir is a functional option that adds a Dir
// property to the underlying Cmd. Dir is the directory
// to execute the command in
func SetCmdDir(dir string) func(MonitoredCmd) MonitoredCmd {
	return func(m MonitoredCmd) MonitoredCmd {
		m.Cmd.Dir = dir
		return m
	}
}

// Run will run the underlying command
// If a termination signal is sent to its signalChan, the
// process will be killed
// TODO the done channel?? will expose the completion status of the process.. somehow
func (m MonitoredCmd) Run() error {
	// when the function returns, write to the done channel to cleanup goroutines
	defer func() {
		fmt.Println("writing to doneness channels")
		m.done <- true
		m.coordinator.SyncChan <- true
	}()

	// start the command's execution
	if err := m.Cmd.Start(); err != nil {
		fmt.Println("error starting command", err)
		return errors.Wrap(err, "failed to start command")
	}

	// go routine that will listen for either an interrupt signal or the command to end naturally
	go func() {
		select {
		case sig := <-m.signalChan:
			syscall.Kill(-m.Cmd.Process.Pid, sig)
			log.Println("command was interrupted")
		case <-m.done:
			log.Println("command ended naturally")
			close(m.done)
		}
	}()

	err := m.Cmd.Wait()
	fmt.Println("cmd result", err)
	return err
}

// Interrupt will send an interrupt signal to the process
func (m MonitoredCmd) Interrupt() {
	m.signalChan <- syscall.SIGINT
	// todo add a way to stall until the process is actually over? can you call cmd.Wait() now??
	return
}
