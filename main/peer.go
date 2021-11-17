package main

import (
	"flag"
	"fmt"
	"os"

	fb "github.com/SimpleFabric/fabric"
)

var (
	org      string
	peerid   string
	isprpeer bool
)

func init() {
	flag.StringVar(&org, "o", "", "the peer's organization")
	flag.StringVar(&peerid, "i", "", "the peer's id")
	flag.BoolVar(&isprpeer, "p", false, "is primary peer")
}

//输入命令参数，组织名，peer名，是否为主节点
func main() {
	flag.Parse()

	if org == "" || peerid == "" {
		fmt.Fprintf(os.Stderr, "Peer argument loss...\n")
		os.Exit(1)
	}
	fb.LoadConfig()
	p, _ := fb.NewPeer(org, peerid, isprpeer)
	go p.Server()
	select {} //wait
}
