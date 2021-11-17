package main

import (
	"flag"

	fb "github.com/SimpleFabric/fabric"
)

//输入命令参数，组织名，order名
func main() {
	flag.Parse()

	fb.LoadConfig()
	orderorg := "orderorg"
	orderid := "orderer1"
	maxblocksize := 10
	o, _ := fb.NewOrder(orderorg, orderid, maxblocksize)
	go o.Server()
	select {} //wait
}
