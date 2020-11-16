package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// ticker.go is a fake "service" that will log a message every second
// it was used in developing demo.go and the monitor API
func main() {
	var runs int
	if os.Getenv("SKIP") != "" {
		runs, _ = strconv.Atoi(os.Getenv("SKIP"))
	}
	for runs < 30 {
		fmt.Println("tick", runs)
		time.Sleep(time.Second * 1)
		runs++
	}
	fmt.Println("ticker done")
}
