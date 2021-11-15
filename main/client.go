package main

import (
	"flag"
	"fmt"
	"math/rand"
	"strconv"

	fb "github.com/SimpleFabric/fabric"
)

var (
	transType  int
	accountNum int
	txNum      int
)
var UserGroupA = make([]string, 5000)
var UserGroupB = make([]string, 5000)

func init() {
	flag.IntVar(&transType, "t", 1, "the transaction type")
	flag.IntVar(&txNum, "n", 0, "the transaction num")
}

func InitUser() {
	for i := 0; i < len(UserGroupA); i++ {
		UserGroupA[i] = "UserA" + strconv.Itoa(i)
		UserGroupB[i] = "UserB" + strconv.Itoa(i)
	}
}

//输入参数，交易类型（1-添加账户，2-转账），发送交易数量
func main() {
	flag.Parse()
	fb.LoadConfig()

	realTxNum := txNum
	if transType == 1 {
		realTxNum *= 2
	}
	InitUser()
	var TxResultchan chan bool = make(chan bool, 100)
	var succSum int = 0
	var totalSum int = 0
	for i := 0; i < txNum; i++ {
		Identity := "client" + strconv.Itoa(i)
		switch transType {
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
