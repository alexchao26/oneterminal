// Package cmdsync has logic for synchronizing multiple shell comands.
// Commands can depend on the completion or readiness of other commands where
// readiness can be determined by the output matching some regular expression.
package cmdsync

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/alexchao26/oneterminal/color"
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
	color         color.Color
	silenceOutput bool
	ready         bool           // if command's dependent's can begin
	readyPattern  *regexp.Regexp // pattern to match against command outputs
	dependsOn     []string       // names of other ShellCmds
	stdout        io.Writer      // set to os.Stdout, included for testing
}

type ShellCmdOption func(*ShellCmd) error

// NewShellCmd defaults to using zsh. bash and sh are also supported
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
		return nil, fmt.Errorf("%q shell not supported. Use zsh|bash|sh", shell)
	}

	execCmd := exec.Command(shell, "-c", command)
	// inherit process group ID's so syscall.Kill reaches ALL child processes
	// https://bigkevmcd.github.io/go/pgrp/context/2019/02/19/terminating-processes-in-go.html
	execCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	s := &ShellCmd{
		command: execCmd,
		stdout:  os.Stdout,
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
	return s.RunContext(context.Background())
}

// RunContext is the same as Run but cancels if the ctx cancels
func (s *ShellCmd) RunContext(ctx context.Context) error {
	// start the command's execution
	if err := s.command.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// make waiting for cmd to run concurrent so select can be used
	done := make(chan error, 1)
	go func() {
		done <- s.command.Wait()
	}()

	var err error
	// blocks until underlying process is done/exits or ctx is done
	select {
	case <-ctx.Done():
		err = ctx.Err()
		s.Interrupt()
	case doneErr := <-done:
		err = doneErr
	}
	s.ready = true
	return err
}

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
		return fmt.Errorf("sending interrupt to %s: %w", s.name, err)
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
		_, err = s.stdout.Write(in)
	} else {
		// if command's name is set, print with prefixed outputs
		prefixed := prefixEveryline(string(in), s.color.Add(s.name))
		_, err = s.stdout.Write([]byte(prefixed))
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
			return fmt.Errorf("directory %q does not exist: %s", dir, err)
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

// Name is a functional option that sets a monitored command's name,
// which is used to prefix each line written to Stdout
func Name(name string) ShellCmdOption {
	return func(s *ShellCmd) error {
		s.name = name
		return nil
	}
}

// Color is a functional option that sets the ansiColor for the outputs
func Color(c color.Color) ShellCmdOption {
	return func(s *ShellCmd) error {
		s.color = c
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
			return fmt.Errorf("compiling regexp %q: %w", pattern, err)
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
// ShellCmd's configs (because it would cause a circular dependency). Some, but
// not all possible config errors are checked at runtime.
func DependsOn(cmdNames ...string) ShellCmdOption {
	return func(s *ShellCmd) error {
		if len(cmdNames) == 0 {
			return fmt.Errorf("zero-length DependsOn list")
		}
		s.dependsOn = cmdNames
		return nil
	}
}

// Environment is a functional option that adds export commands to the start
// of a command. This is a bit of a hacky workaround to maintain exec.ShellCmd's
// default environment, while being able to set additional variables
func Environment(envMap map[string]string) ShellCmdOption {
	var exportVars string
	for k, v := range envMap {
		exportVars += fmt.Sprintf("export %s=%s && ", k, v)
	}

	return func(s *ShellCmd) error {
		s.command.Args[2] = exportVars + s.command.Args[2]
		return nil
	}
}
