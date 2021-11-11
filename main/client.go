package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"

	fb "github.com/SimpleFabric/fabric"
)

var UserGroupA = make([]string, 100)
var UserGroupB = make([]string, 100)

func InitUser() {
	for i := 0; i < len(UserGroupA); i++ {
		UserGroupA[i] = "UserA" + strconv.Itoa(i)
		UserGroupB[i] = "UserB" + strconv.Itoa(i)

	}
}

//输入参数，交易类型（1-添加账户，2-转账），发送交易数量
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Client argument loss...\n")
		os.Exit(1)
	}
	typeNum, _ := strconv.Atoi(os.Args[1])
	txNum, _ := strconv.Atoi(os.Args[2])
	realTxNum := txNum

	if typeNum == 1 {
		realTxNum *= 2
	}

	InitUser()
	var TxResultchan chan bool = make(chan bool, 100)
	var succSum int = 0
	var totalSum int = 0
	for i := 0; i < txNum; i++ {
		Identity := "client" + strconv.Itoa(i)
		switch typeNum {
		case 1:
			ArgsA := [3]string{UserGroupA[i], strconv.Itoa(rand.Intn(100) * 100)}
			go fb.Client(Identity, TxResultchan, "createAccount", ArgsA)
			ArgsB := [3]string{UserGroupB[i], strconv.Itoa(rand.Intn(100) * 100)}
			go fb.Client(Identity, TxResultchan, "createAccount", ArgsB)
		case 2:
			Args := [3]string{UserGroupA[i], UserGroupB[i], strconv.Itoa(rand.Intn(100) * 100)}
			go fb.Client(Identity, TxResultchan, "transfer", Args)
		}
	}
	for {
		select {
		case isSuccess := <-TxResultchan:
			if isSuccess {
				succSum++
			}
			totalSum++
			fmt.Println("The total Tx is ", totalSum, " The success Tx is ", succSum)
		}
		if totalSum == realTxNum {
			break
		}
	}
	fmt.Println("The total Tx is ", totalSum, " The success Tx is ", succSum)
}
