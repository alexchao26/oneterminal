package main

import (
	"fmt"
	"os"

	"github.com/alexchao26/oneterminal/internal/cli"
)

// variable is updated by goreleaser automatically
var version = "dev"

func main() {
	rootCmd, err := cli.Init(version)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = rootCmd.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
