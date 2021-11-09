package main

import (
	"flag"

	fb "github.com/SimpleFabric/fabric"
)

var (
	ordercmd     string
	orderorg     string
	orderid      string
	maxblocksize int
)

func init() {
	// flag.StringVar(&ordercmd, "c", "", "input command, must be order")
	// flag.StringVar(&orderorg, "o", "", "the order's organization")
	//flag.StringVar(&orderid, "i", "", "the order's id")
	//flag.IntVar(&maxblocksize, "m", 100, "the max block size")
}

//输入命令参数，组织名，order名
func main() {
	flag.Parse()

	// if ordercmd != "orderer" {
	// 	fmt.Fprintf(os.Stderr, "The first flag must be orderer...\n")
	// 	os.Exit(1)
	// }

	// if orderorg == "" || orderid == "" {
	// 	fmt.Fprintf(os.Stderr, "Orderer argument loss...\n")
	// 	os.Exit(1)
	// }
	orderorg := "orderorg"
	orderid := "orderer1"
	maxblocksize := 10
	o, _ := fb.NewOrder(orderorg, orderid, maxblocksize)
	go o.Server()
	select {} //wait
}
