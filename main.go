package main

import "github.com/alexchao26/oneterminal/cmd"

func main() {
	cmd.Execute()
}

/*
package main

import (
	"fmt"
	"os/exec"
)

func main() {
	cmd := exec.Command("git", "status")
	if cmd.Stderr != nil {
		fmt.Println("cmd has std err")
		cmd.Output()
	}
	// errStream := bytes.Buffer{}
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("error running cmd", err)
		return
	}
	fmt.Println(string(output))
}

*/
