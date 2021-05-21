package main

import (
	"fmt"
	"os"

	"github.com/alexchao26/oneterminal/internal/cli"
)

// variable is updated by goreleaser automatically but manually added here too
// for manual/from source builds like `go get/install`
var version = "0.5.0"

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
