package main

import (
	"fmt"

	"github.com/nuttapp/pinghist/ping"
)

func main() {
	pr, err := ping.Ping("127.0.0.1")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%#v", pr)

	fmt.Println("END")
}
