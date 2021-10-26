package fabric

import (
	"fmt"
	"log"
	"net/rpc"
)

//当前默认所有交易为写交易
// const NULL_TX int = 0
// const WRITE_TX int = 1
// const READ_TX int = 2

//交易提案，应该包含交易调用的函数和参数，身份信息
type TransProposal struct {
}

type ReadItem struct {
}

type WriteItem struct {
}
type RWSet struct {
	ReadSet  []ReadItem
	WriteSet []WriteItem
}

//交易，应该包含交易的读写集，身份信息。
type Transaction struct {
}

func NewTxProposal(funName string, args [][2]string, identity string) (TransProposal, error) {

}

func SendProposal(org string, peerID string, txp TransProposal) error {

}

func NewTx()

func SendTx()

//发送一笔写交易,并注册监听事件
func Client() error {

}

func call(org string, peerid string, rpcname string, args interface{}, reply interface{}) bool {
	//c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := peerSock(org, peerid)
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
