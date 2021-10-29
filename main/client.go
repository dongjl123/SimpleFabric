package main

import (
	"fmt"
	"os"
	"strconv"

	fb "../fabric"
)

//输入参数，发送交易数量
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Client argument loss...\n")
		os.Exit(1)
	}
	txNum, _ := strconv.Atoi(os.Args[1])
	var TxResultchan chan bool = make(chan bool)
	var succSum int = 0
	var totalSum int = 0
	for i := 0; i < txNum; i++ {
		go fb.Client(i, TxResultchan)
	}
	for {
		select {
		case isSuccess := <-TxResultchan:
			if isSuccess {
				succSum++
			}
			totalSum++
		}
		if totalSum == txNum {
			break
		}
	}
	fmt.Println("The total Tx is ", txNum, " The success Tx is ", succSum)
}
