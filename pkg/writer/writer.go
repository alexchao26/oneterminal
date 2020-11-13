package writer

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
)

// PrefixedStdout satisfies the io.Writer interface.
// The prefix argument will be used at the start of every line that is written
// using Stdout's the Write method
type PrefixedStdout struct {
	prefix string
	stdout io.Writer
}

// NewPrefixedStdout returns a new instance of PrefixedStdout which can be used
// as a stand in for Stdout
func NewPrefixedStdout(prefix string) *PrefixedStdout {
	return &PrefixedStdout{
		prefix: prefix,
		stdout: os.Stdout,
	}
}

// Write helps satisfy the io.Writer interface
// under the hood it adds a prefix to every new line
// it is naive and replaces adds the prefix at every
// newline character
func (p *PrefixedStdout) Write(bytes []byte) (int, error) {
	prefixedString := prefixEveryline(string(bytes), p.prefix)

	_, err := p.stdout.Write([]byte(prefixedString))
	if err != nil {
		return 0, errors.Wrap(err, "writing to os.Stdout")
	}

	// length of input bytes slice must be the same as output int
	// otherwise "short write" error will be thrown
	return len(bytes), err
}

// prefixEachLine adds a given prefix with a bar/pipe "|" to each newline
// of a given string
func prefixEveryline(in, prefix string) (out string) {
	lines := strings.Split(in, "\n")

	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return prefix + " | " + strings.Join(lines, fmt.Sprintf("\n%s | ", prefix)) + "\n"
}
