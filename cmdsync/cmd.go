package cmdsync

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// Cmd is a wrapper around exec.Cmd that eases syncing to other Cmd's via Group.
//
// Its implementation calls the shell directly (through zsh/bash)
//
// Cmd can indicate that the underlying process has reached a "ready state" by
//     1. Its stdout/stderr outputs matching a given regexp.
//     2. Its underlying process completing/exiting with a non-zero code.
//
// An interrupt signal can be sent to the underlying process via Interrupt().
type Cmd struct {
	command       *exec.Cmd
	name          string
	ansiColor     string
	silenceOutput bool
	ready         bool           // if command's dependent's can begin
	readyPattern  *regexp.Regexp // pattern to match against command outputs
	dependsOn     []string
}

type CmdOption func(Cmd) Cmd

// NewCmd makes a command that can be interrupted
// Default shell used is zsh, use functional options to change
// e.g. monitor.NewCmd("echo hello", monitor.SetBashShell)
func NewCmd(command string, options ...CmdOption) *Cmd {
	execCmd := exec.Command("zsh", "-c", command)

	c := Cmd{
		command: execCmd,
	}

	// apply functional options
	for _, f := range options {
		c = f(c)
	}

	execCmd.Stdout = &c
	execCmd.Stderr = &c

	return &c
}

// Run the underlying command. This function blocks until the command exits
func (c *Cmd) Run() error {
	// start the command's execution
	if err := c.command.Start(); err != nil {
		return errors.Wrap(err, "failed to start command")
	}

	// blocks until underlying process is done/exits
	err := c.command.Wait()
	c.ready = true
	return err
}

// TODO add RunContext method for another synchronization option

// Interrupt will send an interrupt signal to the process
func (c *Cmd) Interrupt() error {
	// Process has not started yet
	if c.command.Process == nil || c.command.ProcessState == nil {
		return nil
	}
	if c.command.ProcessState.Exited() {
		return nil
	}
	// Note: if the underlying process does not handle interrupt signals,
	// it will probably just keep running
	err := c.command.Process.Signal(os.Interrupt)
	if err != nil {
		return errors.Wrapf(err, "Error sending interrupt to %s", c.name)
	}
	return nil
}

// Write implements io.Writer, so that Cmd itself can be used for
// exec.Cmd.Stdout and Stderr
// Write "intercepts" writes to Stdout/Stderr to check if the outputs match a
// regexp and determines if a command has reached its "ready state"
// the ready state is used by Orchestrator to coordinate dependent commands
func (c *Cmd) Write(in []byte) (int, error) {
	if c.readyPattern != nil && c.readyPattern.Match(in) {
		c.ready = true
	}

	if c.silenceOutput {
		return len(in), nil
	}

	// if no name is set, just write straight to stdout
	var err error
	if c.name == "" {
		_, err = os.Stdout.Write(in)
	} else {
		// if command's name is set, print with prefixed outputs
		prefixed := prefixEveryline(string(in), fmt.Sprintf("%s%s%s", c.ansiColor, c.name, "\033[0m"))
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
func (c *Cmd) IsReady() bool {
	return c.ready
}

// SetBashShell is a functional option to change the executing shell to zsh
func SetBashShell(c Cmd) Cmd {
	c.command.Args[0] = "bash"
	resolvedPath, err := exec.LookPath("bash")
	if err != nil {
		panic(fmt.Sprintf("Error setting bash as shell %s", err))
	}

	c.command.Path = resolvedPath
	return c
}

// SetCmdDir is a functional option that adds a Dir property to the underlying
// command. Dir is the directory to execute the command from
func SetCmdDir(dir string) CmdOption {
	return func(c Cmd) Cmd {
		if dir[0] == '~' {
			dir = fmt.Sprintf("$HOME%s", dir[1:])
		}
		expandedDir := os.ExpandEnv(dir)

		_, err := os.Stat(expandedDir)
		if os.IsNotExist(err) {
			panic(fmt.Sprintf("Directory %q does not exist: %s", dir, err))
		}

		c.command.Dir = expandedDir
		return c
	}
}

// SetSilenceOutput sets the command's Stdout and Stderr to nil so no output
// will be seen in the terminal
func SetSilenceOutput(c Cmd) Cmd {
	c.silenceOutput = true
	return c
}

// SetCmdName is a functional option that sets a monitored command's name,
// which is used to prefix each line written to Stdout
func SetCmdName(name string) CmdOption {
	return func(c Cmd) Cmd {
		c.name = name
		return c
	}
}

// SetColor is a functional option that sets the ansiColor for the outputs
func SetColor(terminalColor string) CmdOption {
	return func(c Cmd) Cmd {
		c.ansiColor = terminalColor
		return c
	}
}

// SetReadyPattern is a functional option that takes in a pattern string
// that must compile into a valid regexp and sets it to monitored command's
// readyPattern field
func SetReadyPattern(pattern string) CmdOption {
	return func(c Cmd) Cmd {
		c.readyPattern = regexp.MustCompile(pattern)
		return c
	}
}

// SetDependsOn is a functional option that sets a slice of dependencies
// for this command. The dependencies are names of commands that need to be done
// or ready prior to this command starting
func SetDependsOn(cmdNames []string) CmdOption {
	return func(c Cmd) Cmd {
		c.dependsOn = cmdNames
		return c
	}
}

// SetEnvironment is a functional option that adds export commands to the start
// of a command. This is a bit of a hacky workaround to maintain exec.Cmd's
// default environment, while being able to set additional variables
func SetEnvironment(envMap map[string]string) CmdOption {
	var envSlice []string
	for k, v := range envMap {
		envSlice = append(envSlice, k+"="+v)
	}

	exportString := "export " + strings.Join(envSlice, " && export ") + " && "
	return func(c Cmd) Cmd {
		c.command.Args[2] = exportString + c.command.Args[2]
		return c
	}
}
