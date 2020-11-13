# oneterminal to rule them all!

A configurable CLI to run multiple terminal windows within a single command.

# Overview

onetermainl is a CLI written in [Go](https://golang.org/). Each command is configured in a YAML file stored in the ~/.config/oneterminal directory. For more details on how to configure a command, refer to [example configurations](#example-configurations).

It has been built and tested on macOS with the zsh shell, but should work for bash as well (and maybe linux?).


# How to Install

If you have a local [Go installation](https://golang.org/doc/install):
```go
go get github.com/alexchao26/oneterminal/cmd/oneterminal
```

Alternatively, without a local Go installation, you can download the binary directly from the [Github releases](https://github.com/alexchao26/oneterminal/releases).
Move this file into somewhere in your $PATH environment variable and make it executable.
```shell
# make executable
$ chmod -x /path/to/oneterminal

# To view your path variable
$ echo $PATH

# potentially move it to /usr/local/bin
$ mv /path/to/oneterminal /usr/local/bin
```

Note: Installation via brew is in the works.


# Example Configurations

## Utilizing the example config generator
`oneterminal example` will create a config at ~/.config/oneterminal/example.yml containing helpful comments about each yaml field
```yaml
# The name of the command, it cannot have special characters
name: somename

# shell to use, zsh and bash are supported
shell: zsh

# a short description of what this command does
short: an example command that says hello twice
# OPTIONAL: longer description of what this command does
long: Optional longer description

# commands are made of
#   1. command string (the command to run, will be expanded via os.ExpandEnv)
#   2. name string, text to prefix each line of this command's output
#      NOTE: an empty string is a valid name and is useful for things like vault
#            which write to stdout in small chunks
#   3. directory string (optional), what directory to run the command from
#      NOTE: use $HOME, not ~. This strings gets passed through os.ExpandEnv
#   4. silence boolean (optional: default false), if true will silence that command's output
commands:
- name: greeter-1
  command: echo hello from window 1
  directory: $HOME/go
  silence: false
- name: greeter-2
  command: echo hello from window 2
  silence: false
- name: ""
  command: echo "they silenced me :'("
  silence: true
```

Run `oneterminal help` to see this command show up under available commands. Note that this command's name is set by the name field in example.yml.

# oneterminal Commands

`oneterminal example`: Makes a demo oneterminal config in ~/.config/oneterminal

`oneterminal completion --help`: Get helper text to setup shell completion for zsh or bash shells

`oneterminal help`: Help about any command

`oneterminal <your-configured-commands>`

# Contributing to oneterminal

This project is still in its infancy and its future path is undetermined.

I welcome contributions but ask that you open an issue to discuss bugs and desired features!

# Dependencies

- github.com/pkg/errors v0.8.1
- github.com/spf13/cobra v1.1.1
- gopkg.in/yaml.v2 v2.3.0
