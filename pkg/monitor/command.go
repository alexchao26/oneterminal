package monitor

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
)

// MonitoredCmd is a wrapper around exec.command
// its implementations calls the shell directly and
// sending the process a termination/interruption signal
// and checking if the process has completed
type MonitoredCmd struct {
	command    *exec.Cmd           // underlying command to run
	signalChan chan syscall.Signal // channel that receives any interrupt signals
	done       chan bool           // channel to cleanup when the command finishes
}

// NewMonitoredCmd makes a command that can be interrupted
// via its signalChan channel, which is exposed by its
// Interrupt method
// Default shell used is zsh, use functional options to change
// e.g. monitor.NewMonitoredCmd("echo hello", monitor.BashShell)
func NewMonitoredCmd(command string, options ...func(MonitoredCmd) MonitoredCmd) MonitoredCmd {
	c := exec.Command("zsh", "-c", command)
	c.Stdout = os.Stdout
	c.Stderr = os.Stdout
	// SysProcAttr sets the child process's PID to the parent's PID
	// making the process identifiable if it needs to be interrupted
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	m := MonitoredCmd{
		command:    c,
		signalChan: make(chan syscall.Signal, 1),
		done:       make(chan bool, 1),
	}

	// apply functional options
	for _, f := range options {
		m = f(m)
	}

	return m
}

// BashShell is a functional option to change the executing shell to zsh
func BashShell(m MonitoredCmd) MonitoredCmd {
	m.command.Args[0] = "bash"
	resolvedPath, err := exec.LookPath("bash")
	if err != nil {
		panic(fmt.Sprintf("Error setting bash as shell %s", err))
	}

	m.command.Path = resolvedPath
	return m
}

// SetCmdDir is a functional option that adds a Dir
// property to the underlying command. Dir is the directory
// to execute the command in
func SetCmdDir(dir string) func(MonitoredCmd) MonitoredCmd {
	return func(m MonitoredCmd) MonitoredCmd {
		expandedDir := os.ExpandEnv(dir)
		if _, err := os.Stat(expandedDir); os.IsNotExist(err) {
			panic(fmt.Sprintf("Directory does not exist %s\nNOTE: use $HOME, not ~", err))
		}

		m.command.Dir = expandedDir
		return m
	}
}

// SilenceOutput sets the command's Stdout and Stderr to nil
// so no output will be seen in the terminal
func SilenceOutput(m MonitoredCmd) MonitoredCmd {
	m.command.Stdout = nil
	m.command.Stderr = nil
	return m
}

// Run will run the underlying command. This function is blocking
// until the command is done or is interrupted
// It can be interrupted via the Interrupt receiver method
func (m MonitoredCmd) Run() error {
	// when the function returns, write to the done channel to cleanup goroutines
	defer func() {
		m.done <- true
	}()

	// start the command's execution
	if err := m.command.Start(); err != nil {
		return errors.Wrap(err, "failed to start command")
	}

	// go routine that will listen for either an interrupt signal or the command to end naturally
	go func() {
		select {
		case sig := <-m.signalChan:
			syscall.Kill(-m.command.Process.Pid, sig)
		case <-m.done:
			// close channels
			close(m.done)
			close(m.signalChan)
		}
	}()

	return m.command.Wait()
}

// Interrupt will send an interrupt signal to the process
func (m MonitoredCmd) Interrupt() {
	m.signalChan <- syscall.SIGINT
	return
}

// TODO add stdout wrapper
