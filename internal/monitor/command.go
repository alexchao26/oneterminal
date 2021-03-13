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

// MonitoredCmd is a wrapper around exec.Cmd
//
// Its implementation calls the shell directly (through zsh/bash)
//
// MonitoredCmd can indicate that the underlying process is ready by either
// matching a regexp or if the underlying process exits.
// An interrupt signal can be sent to the underlying process via Interrupt().
type MonitoredCmd struct {
	command       *exec.Cmd
	name          string
	ansiColor     string
	silenceOutput bool
	ready         bool           // if command's dependent's can begin
	readyPattern  *regexp.Regexp // pattern to match against command outputs
	dependsOn     []string
}

type MonitoredCmdOption func(MonitoredCmd) MonitoredCmd

// NewMonitoredCmd makes a command that can be interrupted
// Default shell used is zsh, use functional options to change
// e.g. monitor.NewMonitoredCmd("echo hello", monitor.SetBashShell)
func NewMonitoredCmd(command string, options ...MonitoredCmdOption) *MonitoredCmd {
	c := exec.Command("zsh", "-c", command)

	m := MonitoredCmd{
		command: c,
	}

	// apply functional options
	for _, f := range options {
		m = f(m)
	}

	c.Stdout = &m
	c.Stderr = &m

	return &m
}

// Run the underlying command. This function blocks until the command exits
func (m *MonitoredCmd) Run() error {
	// start the command's execution
	if err := m.command.Start(); err != nil {
		return errors.Wrap(err, "failed to start command")
	}

	// blocks until underlying process is done/exits
	err := m.command.Wait()
	m.ready = true
	return err
}

// TODO add RunContext method for another synchronization option

// Interrupt will send an interrupt signal to the process
func (m *MonitoredCmd) Interrupt() {
	// Process has not started yet
	if m.command.Process == nil || m.command.ProcessState == nil {
		return
	}
	if m.command.ProcessState.Exited() {
		return
	}
	// Note: if the underlying process does not handle interrupt signals,
	// it will probably just keep running
	err := m.command.Process.Signal(syscall.SIGINT)
	if err != nil {
		fmt.Printf("Error sending interrupt to %s: %v\n", m.name, err)
	}
}

// Write implements io.Writer, so that MonitoredCmd itself can be used for
// exec.Cmd.Stdout and Stderr
// Write "intercepts" writes to Stdout/Stderr to check if the outputs match a
// regexp and determines if a command has reached its "ready state"
// the ready state is used by Orchestrator to coordinate dependent commands
func (m *MonitoredCmd) Write(in []byte) (int, error) {
	if m.readyPattern != nil && m.readyPattern.Match(in) {
		m.ready = true
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

// SetCmdDir is a functional option that adds a Dir property to the underlying
// command. Dir is the directory to execute the command from
func SetCmdDir(dir string) MonitoredCmdOption {
	return func(m MonitoredCmd) MonitoredCmd {
		expandedDir := os.ExpandEnv(dir)
		if _, err := os.Stat(expandedDir); os.IsNotExist(err) {
			panic(fmt.Sprintf("Directory does not exist %s\nNOTE: use $HOME, not ~", err))
		}

		m.command.Dir = expandedDir
		return m
	}
}

// SetSilenceOutput sets the command's Stdout and Stderr to nil so no output
// will be seen in the terminal
func SetSilenceOutput(m MonitoredCmd) MonitoredCmd {
	m.silenceOutput = true
	return m
}

// SetCmdName is a functional option that sets a monitored command's name,
// which is used to prefix each line written to Stdout
func SetCmdName(name string) MonitoredCmdOption {
	return func(m MonitoredCmd) MonitoredCmd {
		m.name = name
		return m
	}
}

// SetColor is a functional option that sets the ansiColor for the outputs
func SetColor(terminalColor string) MonitoredCmdOption {
	return func(m MonitoredCmd) MonitoredCmd {
		m.ansiColor = terminalColor
		return m
	}
}

// SetReadyPattern is a functional option that takes in a pattern string
// that must compile into a valid regexp and sets it to monitored command's
// readyPattern field
func SetReadyPattern(pattern string) MonitoredCmdOption {
	return func(m MonitoredCmd) MonitoredCmd {
		m.readyPattern = regexp.MustCompile(pattern)
		return m
	}
}

// SetDependsOn is a functional option that sets a slice of dependencies
// for this command. The dependencies are names of commands that need to be done
// or ready prior to this command starting
func SetDependsOn(cmdNames []string) MonitoredCmdOption {
	return func(m MonitoredCmd) MonitoredCmd {
		m.dependsOn = cmdNames
		return m
	}
}

// SetEnvironment is a functional option that adds export commands to the start
// of a command. This is a bit of a hacky workaround to maintain exec.Cmd's
// default environment, while being able to set additional variables
func SetEnvironment(envMap map[string]string) MonitoredCmdOption {
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
