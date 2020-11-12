package writer

import (
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/pkg/errors"
)

// PrefixedStdout satisfies the io.Writer interface.
// The prefix argument will be used at the start of every line that is written
// using Stdout's the Write method
type PrefixedStdout struct {
	prefix string
	stdout io.Writer
}

// NewPrefixedStdout returns a new instance of PrefixedStdout
// TODO add to yaml config for optional prefix name
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

// TODO add test
func prefixEveryline(in, prefix string) (out string) {
	newLineRegex := regexp.MustCompile("\n")

	// add the prefix and a space after all newlines
	prefixedString := newLineRegex.ReplaceAllString(in, fmt.Sprintf("\n%s ", prefix))

	// add prefix onto start, remove last prefix from end (5 characters)
	prefixedString = fmt.Sprintf("%s %s", prefix, prefixedString[:len(prefixedString)-5])

	return prefixedString
}
