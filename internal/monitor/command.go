package monitor

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

// MonitoredCmd is a wrapper around exec.command
// its implementations calls the shell directly and
// sending the process a termination/interruption signal
// and checking if the process has completed
type MonitoredCmd struct {
	command       *exec.Cmd
	name          string
	ansiColor     string
	silenceOutput bool
	signalChan    chan syscall.Signal
	ready         bool           // if command's dependent's can begin
	readyPattern  *regexp.Regexp // pattern to match against command outputs
	readyChan     chan bool      // channel needed to get around no pointer receiver on Write method
	dependsOn     []string
}

// NewMonitoredCmd makes a command that can be interrupted
// via its signalChan channel, which is exposed by its
// Interrupt method
// Default shell used is zsh, use functional options to change
// e.g. monitor.NewMonitoredCmd("echo hello", monitor.SetBashShell)
func NewMonitoredCmd(command string, options ...func(MonitoredCmd) MonitoredCmd) MonitoredCmd {
	c := exec.Command("zsh", "-c", command)

	// SysProcAttr sets the child process's PID to the parent's PID
	// making the process identifiable if it needs to be interrupted
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	m := MonitoredCmd{
		command:    c,
		signalChan: make(chan syscall.Signal, 1),
		readyChan:  make(chan bool, 1),
	}

	// apply functional options
	for _, f := range options {
		m = f(m)
	}

	// Stdout and Stderr are set to the MonitoredCmd (which satisfies io.Writer)
	// to intercept outputs and determine if a command has reached its "ready
	// state"
	c.Stdout = m
	c.Stderr = m

	return m
}

// Run will run the underlying command. This function is blocking
// until the command is done or is interrupted
// It can be interrupted via the Interrupt receiver method
func (m *MonitoredCmd) Run() error {
	// channel to cleanup goroutines if command completes
	done := make(chan bool, 1)

	// start the command's execution
	if err := m.command.Start(); err != nil {
		return errors.Wrap(err, "failed to start command")
	}

	// listen for either an interrupt signal or the command to end naturally
	go func() {
		select {
		case sig := <-m.signalChan:
			syscall.Kill(-m.command.Process.Pid, sig)
		case <-done:
		}
	}()

	go func() {
		// read in an infinite loop so Write's regexp check does not block
		for {
			<-m.readyChan
			m.ready = true
		}
	}()

	err := m.command.Wait()
	m.readyChan <- true
	done <- true
	return err
}

// Interrupt will send an interrupt signal to the process
func (m MonitoredCmd) Interrupt() {
	m.signalChan <- syscall.SIGINT
}

// Write satisfies the Writer interface, so that MonitoredCmd itself can be used
// for exec.Cmc.Stdout and Stderr
// Write "intercepts" writes to Stdout/Stderr to check if the outputs match a
// regexp and determines if a command has reached its "ready state"
// the ready state is used elsewhere coordinate dependent commands
func (m MonitoredCmd) Write(in []byte) (int, error) {
	if m.readyPattern != nil && m.readyPattern.Match(in) {
		m.readyChan <- true
	}

	if m.silenceOutput {
		return len(in), nil
	}

	// if no name is set, just write straight to stdout
	var err error
	if m.name == "" {
		_, err = os.Stdout.Write(in)
	} else {
		// if command's name is set, print with prefixed outputs
		prefixed := prefixEveryline(string(in), fmt.Sprintf("%s%s%s", m.ansiColor, m.name, "\033[0m"))
		_, err = os.Stdout.Write([]byte(prefixed))
	}

	return len(in), err
}

// prefixEachLine adds a given prefix with a bar/pipe " | " to each newline
func prefixEveryline(in, prefix string) (out string) {
	lines := strings.Split(in, "\n")

	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return prefix + " | " + strings.Join(lines, fmt.Sprintf("\n%s | ", prefix)) + "\n"
}

// IsReady is a simple getter for the ready state of a monitored command
func (m *MonitoredCmd) IsReady() bool {
	return m.ready
}

// SetBashShell is a functional option to change the executing shell to zsh
func SetBashShell(m MonitoredCmd) MonitoredCmd {
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

// SetSilenceOutput sets the command's Stdout and Stderr to nil
// so no output will be seen in the terminal
func SetSilenceOutput(m MonitoredCmd) MonitoredCmd {
	m.silenceOutput = true
	return m
}

// SetCmdName is a functional option that sets a monitored command's name,
// which is used to prefix each line written to Stdout
func SetCmdName(name string) func(MonitoredCmd) MonitoredCmd {
	return func(m MonitoredCmd) MonitoredCmd {
		m.name = name
		return m
	}
}

// SetColor is a functional option that sets the ansiColor for the outputs
func SetColor(terminalColor string) func(MonitoredCmd) MonitoredCmd {
	return func(m MonitoredCmd) MonitoredCmd {
		m.ansiColor = terminalColor
		return m
	}
}

// SetReadyPattern is a functional option that takes in a pattern string
// that must compile into a valid regexp and sets it to monitored command's
// readyPattern field
func SetReadyPattern(pattern string) func(MonitoredCmd) MonitoredCmd {
	return func(m MonitoredCmd) MonitoredCmd {
		m.readyPattern = regexp.MustCompile(pattern)
		return m
	}
}

// SetDependsOn is a functional option that sets a slice of dependencies
// for this command. The dependencies are names of commands that need to be done
// or ready prior to this command starting
func SetDependsOn(cmdNames []string) func(MonitoredCmd) MonitoredCmd {
	return func(m MonitoredCmd) MonitoredCmd {
		m.dependsOn = cmdNames
		return m
	}
}

// SetEnvironment is a functional option that adds export commands to the start
// of a command
func SetEnvironment(envMap map[string]string) func(MonitoredCmd) MonitoredCmd {
	var envSlice []string
	for k, v := range envMap {
		envSlice = append(envSlice, k+"="+v)
	}

	exportString := "export " + strings.Join(envSlice, " && export ") + " && "
	return func(m MonitoredCmd) MonitoredCmd {
		m.command.Args[2] = exportString + m.command.Args[2]
		return m
	}
}
