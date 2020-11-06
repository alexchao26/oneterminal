package main

import (
	"fmt"
	"time"
)

func main() {
	var runs int
	for runs < 30 {
		fmt.Println("tick", runs)
		time.Sleep(time.Second * 1)
		runs++
	}
	fmt.Println("ticker done")
}
