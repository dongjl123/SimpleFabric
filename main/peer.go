package main

import (
	fb "../fabric"
	"fmt"
	"os"
)

//输入参数，组织名，peer名
func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Peer argument loss...\n")
		os.Exit(1)
	}
	p, _ := fb.NewPeer(os.Args[1], os.Args[2])
	go p.Server()
	select {} //wait
}
