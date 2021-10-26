package main

import (
	"fmt"
	"os"
	"strconv"

	fb "../fabric"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Peer argument loss...\n")
		os.Exit(1)
	}
	id, _ := strconv.Atoi(os.Args[2])
	p := fb.NewPeer(os.Args[1], id)
	go p.Server()
	select {} //wait
}
