package fabric

import (
	"fmt"
	"log"
	"net/rpc"
	"strconv"
)

//当前默认所有交易为写交易
// const NULL_TX int = 0
// const WRITE_TX int = 1
// const READ_TX int = 2

//默认配置
var orgs = [2]string{"org1", "org2"}
var peers = [2]string{"peerA", "peerB"}

//交易提案，应该包含交易调用的函数和参数，身份信息
type TransProposal struct {
	funName  string
	args     [3]string
	identity string
}

//交易，应该包含交易的读写集，身份信息。
type Transaction struct {
}

func NewTxProposal(funName string, args [3]string, identity string) TransProposal {
	transProposal := TransProposal{funName: funName, args: args, identity: identity}
	return transProposal
}

func SendProposal(org string, peerID string, txp TransProposal) error {
	sendArgs := ProposalArgs{TP: txp}
	sendReply := ProposalReply{}
	return call(org, peerID, "TransProposal", &sendArgs, &sendReply)
}

func NewTx() {

}

func SendTx() {

}

//发送一笔写交易,并注册监听事件
func Client(id int, doneChan chan bool) {
	//生成交易提案，发送交易提案
	identity := "client" + strconv.Itoa(id)
	args := [3]string{"Alice", "Bob", "10"}
	txp := NewTxProposal("transfer", args, identity)
	for i := 0; i < 2; i++ {
		err := SendProposal(orgs[i], peers[0], txp)
		if err != nil {
			fmt.Println("send Proposal fail: ", err)
		}
	}
	//提案搜集完毕，进行校验

	//生成交易，发送交易

	//监听
}

func call(org string, peerid string, rpcname string, args interface{}, reply interface{}) error {
	//c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := peerSock(org, peerid)
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()
	err = c.Call(rpcname, args, reply)
	return err
}
