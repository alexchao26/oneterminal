package cmdsync

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

// ShellCmd is a wrapper around exec.Cmd that eases syncing to other ShellCmd's via Group.
//
// Its implementation calls the shell directly (through zsh/bash)
//
// ShellCmd can indicate that the underlying process has reached a "ready state" by
//     1. Its stdout/stderr outputs matching a given regexp.
//     2. Its underlying process completing/exiting with a non-zero code.
//
// An interrupt signal can be sent to the underlying process via Interrupt().
type ShellCmd struct {
	command       *exec.Cmd
	name          string
	ansiColor     string
	silenceOutput bool
	ready         bool           // if command's dependent's can begin
	readyPattern  *regexp.Regexp // pattern to match against command outputs
	dependsOn     []string
}

type ShellCmdOption func(*ShellCmd) error

// NewCmd defaults to using zsh. bash and sh are also supported
func NewShellCmd(shell, command string, options ...ShellCmdOption) (*ShellCmd, error) {
	if shell == "" {
		shell = "zsh"
	}
	allowedShells := map[string]bool{
		"zsh":  true,
		"bash": true,
		"sh":   true,
	}
	if !allowedShells[shell] {
		return nil, errors.Errorf("%q shell not supported. Use zsh|bash|sh", shell)
	}

	execCmd := exec.Command(shell, "-c", command)
	// inherit process group ID's so syscall.Kill reaches ALL child processes
	// https://bigkevmcd.github.io/go/pgrp/context/2019/02/19/terminating-processes-in-go.html
	execCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	s := &ShellCmd{
		command: execCmd,
	}

	// apply functional options
	for _, opt := range options {
		err := opt(s)
		if err != nil {
			return nil, err
		}
	}

	execCmd.Stdout = s
	execCmd.Stderr = s

	return s, nil
}

// Run the underlying command. This function blocks until the command exits
func (s *ShellCmd) Run() error {
	// start the command's execution
	if err := s.command.Start(); err != nil {
		return errors.Wrap(err, "failed to start command")
	}

	// blocks until underlying process is done/exits
	err := s.command.Wait()
	s.ready = true
	return err
}

// TODO add RunContext method for another synchronization option

// Interrupt will send an interrupt signal to the process
func (s *ShellCmd) Interrupt() error {
	// Process is not set if it has not been started yet
	if s.command == nil || s.command.Process == nil {
		return nil
	}

	// send an interrupt to the entire process group to reach "grandchildren"
	// https://bigkevmcd.github.io/go/pgrp/context/2019/02/19/terminating-processes-in-go.html
	// is syscall.SIGINT okay here? might need to be SIGTERM/SIGKILL
	err := syscall.Kill(-s.command.Process.Pid, syscall.SIGINT)
	if err != nil {
		return errors.Wrapf(err, "Error sending interrupt to %s", s.name)
	}
	return nil
}

// Write implements io.Writer, so that ShellCmd itself can be used for
// exec.ShellCmd.Stdout and Stderr
// Write "intercepts" writes to Stdout/Stderr to check if the outputs match a
// regexp and determines if a command has reached its "ready state"
// the ready state is used by Orchestrator to coordinate dependent commands
func (s *ShellCmd) Write(in []byte) (int, error) {
	if s.readyPattern != nil && s.readyPattern.Match(in) {
		s.ready = true
	}

	if s.silenceOutput {
		return len(in), nil
	}

	// if no name is set, just write straight to stdout
	var err error
	if s.name == "" {
		_, err = os.Stdout.Write(in)
	} else {
		// if command's name is set, print with prefixed outputs
		prefixed := prefixEveryline(string(in), fmt.Sprintf("%s%s%s", s.ansiColor, s.name, "\033[0m"))
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
func (s *ShellCmd) IsReady() bool {
	return s.ready
}

// CmdDir is a functional option that modifies the Dir property of the
// underlying exec.ShellCmd which is the directory to execute the Command from
func CmdDir(dir string) ShellCmdOption {
	return func(s *ShellCmd) error {
		// expand '~' to $HOME for os.ExpandEnv to pickup
		if dir[0] == '~' {
			dir = fmt.Sprintf("$HOME%s", dir[1:])
		}
		expandedDir := os.ExpandEnv(dir)

		_, err := os.Stat(expandedDir)
		if os.IsNotExist(err) {
			return errors.Errorf("Directory %q does not exist: %s", dir, err)
		}

		s.command.Dir = expandedDir
		return nil
	}
}

// SilenceOutput sets the command's Stdout and Stderr to nil so no output
// will be seen in the terminal
func SilenceOutput() ShellCmdOption {
	return func(s *ShellCmd) error {
		s.silenceOutput = true
		return nil
	}
}

// CmdName is a functional option that sets a monitored command's name,
// which is used to prefix each line written to Stdout
func CmdName(name string) ShellCmdOption {
	return func(s *ShellCmd) error {
		s.name = name
		return nil
	}
}

// SetColor is a functional option that sets the ansiColor for the outputs
func SetColor(terminalColor string) ShellCmdOption {
	return func(s *ShellCmd) error {
		s.ansiColor = terminalColor
		return nil
	}
}

// ReadyPattern is a functional option that takes in a pattern string
// that must compile into a valid regexp and sets it to monitored command's
// readyPattern field
func ReadyPattern(pattern string) ShellCmdOption {
	return func(s *ShellCmd) error {
		r, err := regexp.Compile(pattern)
		if err != nil {
			return errors.Wrapf(err, "compiling regexp %q", pattern)
		}
		s.readyPattern = r
		return nil
	}
}

// DependsOn is a functional option that sets a slice of dependencies for this
// command. The dependencies are names of commands that need to have completed
// or reached a ready state prior to this command starting.
//
// Note that there is no validation that the cmdNames are valid/match other
// ShellCmd's configs (because it would cause a circular dependency). Some, but not
// all possible config errors are checked at runtime.
func DependsOn(cmdNames []string) ShellCmdOption {
	return func(s *ShellCmd) error {
		s.dependsOn = cmdNames
		return nil
	}
}

// Environment is a functional option that adds export commands to the start
// of a command. This is a bit of a hacky workaround to maintain exec.ShellCmd's
// default environment, while being able to set additional variables
func Environment(envMap map[string]string) ShellCmdOption {
	var envSlice []string
	for k, v := range envMap {
		envSlice = append(envSlice, k+"="+v)
	}

	exportString := "export " + strings.Join(envSlice, " && export ") + " && "
	return func(s *ShellCmd) error {
		s.command.Args[2] = exportString + s.command.Args[2]
		return nil
	}
}
