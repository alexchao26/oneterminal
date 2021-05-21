package cmdsync_test

import (
	"log"

	"github.com/alexchao26/oneterminal/cmdsync"
)

func ExampleShellCmd_Run_simple() {
	cmd, err := cmdsync.NewShellCmd("bash", "echo hello potato")
	if err != nil {
		log.Fatal(err)
	}
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// hello potato
}

func ExampleShellCmd_Run_options() {
	cmd, err := cmdsync.NewShellCmd("bash",
		"echo potato && echo loves $FAV_FOOD",
		cmdsync.Name("monkey"),
		cmdsync.Environment(map[string]string{
			"FAV_FOOD": "cheeseburgers",
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// monkey | potato
	// monkey | loves cheeseburgers
}
