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
	seconds := 10
	if os.Getenv("SECONDS") != "" {
		seconds, _ = strconv.Atoi(os.Getenv("SECONDS"))
	}
	for i := 0; i < seconds; i++ {
		fmt.Println("tick", i)
		time.Sleep(time.Second * 1)
	}
	fmt.Println("ticker done")
}
