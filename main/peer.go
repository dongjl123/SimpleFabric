package main

import (
	fb "../fabric"
	"fmt"
	"os"
	"flag"
)

var(
	cmd string
	org string
	peerid string
	isprpeer bool
)

func init()  {
	flag.StringVar(&cmd, "c", "", "input command, must be peer")
	flag.StringVar(&org, "o", "", "the peer's organization")
	flag.StringVar(&peerid, "i", "", "the peer's id")
	flag.BoolVar(&isprpeer, "p", false, "is primary peer")
}

//输入命令参数，组织名，peer名，是否为主节点
func main() {
	flag.Parse()

	if cmd != "peer"{
		fmt.Fprintf(os.Stderr, "The first flag must be peer...\n")
		os.Exit(1)
	}

	if org == "" || peerid == ""{
		fmt.Fprintf(os.Stderr, "Peer argument loss...\n")
		os.Exit(1)
	}
	p, _ := fb.NewPeer(org, peerid, isprpeer)
	go p.Server()
	select {} //wait
}
