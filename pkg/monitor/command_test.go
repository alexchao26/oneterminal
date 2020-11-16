package monitor

import "testing"

var prefixEachLineTests = []struct {
	input, prefix, want string
}{
	{"hi", "pre-1", "pre-1 | hi\n"},
	{"hello\nasdf", "cmd", "cmd | hello\ncmd | asdf\n"},
	{"Starting...\nWaiting...\nReady!", "launcher", "launcher | Starting...\nlauncher | Waiting...\nlauncher | Ready!\n"},
}

func TestPrefixEachLine(t *testing.T) {
	for _, test := range prefixEachLineTests {
		actual := prefixEveryline(test.input, test.prefix)
		if actual != test.want {
			t.Errorf("Expected %q, got %q", test.want, actual)
		}
	}
}
