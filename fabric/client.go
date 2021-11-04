package fabric

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
	"strconv"
)

//当前默认所有交易为写交易
// const NULL_TX int = 0
// const WRITE_TX int = 1
// const READ_TX int = 2

//默认配置
var orgNum int = 2
var orgs = [2]string{"org1", "org2"}
var peers = [2]string{"peerA", "peerB"}

//交易提案，应该包含交易调用的函数和参数，身份信息
type TransProposal struct {
	funName  string
	args     [3]string
	identity string
}

//交易，应该包含交易ID，交易的读写集，身份信息。
type Transaction struct {
	TxID     string
	identity string
	RWSet
}

func NewTxProposal(funName string, args [3]string, identity string) TransProposal {
	transProposal := TransProposal{funName: funName, args: args, identity: identity}
	return transProposal
}

func SendProposal(org string, peerID string, txp TransProposal) (ProposalReply, error) {
	sendArgs := ProposalArgs{TP: txp}
	sendReply := ProposalReply{}
	err := call(org, peerID, "TransProposal", &sendArgs, &sendReply)
	return sendReply, err
}

func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

func Encode(data interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func makeTxID(id string, data RWSet) (string, error) {
	encodeData, err := Encode(data)
	if err != nil {
		return "", err
	}
	txid := ihash(id + string(encodeData))
	txidString := strconv.Itoa(txid)
	return txidString, nil
}

func NewTx(rwset RWSet, identity string) (Transaction, error) {
	txID, err := makeTxID(identity, rwset)
	if err != nil {
		fmt.Println("'make TxID ERROR: ", err)
		return Transaction{}, err
	}
	newTx := Transaction{RWSet: rwset, identity: identity, TxID: txID}
	return newTx, nil
}

func SendTx(Tx Transaction) (OrderReply, error) {
	ordArgs := OrderArgs{TX: Tx}
	ordReply := OrderReply{}
	err := call("orderorg", "orderer1", "TransOrder", &ordArgs, &ordReply)
	return ordReply, err
}

func ClientRegisterEvent(txid string) (ReEvReply, error) {
	reArgs := ReEvArgs{TxID: txid}
	reReply := ReEvReply{}
	err := call(orgs[0], peers[0], "RegisterEvent", &reArgs, &reReply)
	return reReply, err
}

//背书提案验证函数
func endorserVaildator(RWSlice []RWSet) bool {
	if len((RWSlice)) < orgNum {
		fmt.Println("Endorser response number is not enough")
		return false
	}
	//正常来讲还需要先比较读写集的大小是否相同，这里先略过了，如果报数组越界错误考虑这里
	//比较读写集是否完全相同
	for i, r := range RWSlice[0].ReadSet {
		for _, rwset := range RWSlice {
			if rwset.ReadSet[i] != r {
				fmt.Println("Endorser policy is not satisfied")
				return false
			}
		}
	}
	for i, r := range RWSlice[0].WriteSet {
		for _, rwset := range RWSlice {
			if rwset.WriteSet[i] != r {
				fmt.Println("Endorser policy is not satisfied")
				return false
			}
		}
	}
	return true
}

//发送一笔写交易,并注册监听事件
func Client(id int, doneChan chan bool) {
	//生成交易提案，发送交易提案
	identity := "client" + strconv.Itoa(id)
	args := [3]string{"Alice", "Bob", "10"}
	txp := NewTxProposal("transfer", args, identity)
	RWSlice := []RWSet{}
	for i := 0; i < orgNum; i++ {
		sendReply, err := SendProposal(orgs[i], peers[0], txp)
		if err != nil {
			fmt.Println("send Proposal fail: ", err)
			continue
		}
		if sendReply.IsSuccess == false {
			fmt.Println("ERROR: ", orgs[i], peers[0], " endorser failed")
		} else {
			RWSlice = append(RWSlice, sendReply.RW)
		}
	}
	//提案搜集完毕，进行校验
	if endorserVaildator(RWSlice) == false {
		doneChan <- false
		return
	}
	//生成交易，发送交易
	Tx, err := NewTx(RWSlice[0], identity)
	if err != nil {
		fmt.Println("New TX ERROR")
		doneChan <- false
		return
	}
	sendReply, err := SendTx(Tx)
	if err != nil || sendReply.IsSuccess == false {
		fmt.Println("send Tx fail:", err)
		doneChan <- false
		return
	}
	//监听
	reReply, err := ClientRegisterEvent(identity)
	if err != nil {
		fmt.Println("listen registered event error")
		doneChan <- false
		return
	}
	if reReply.IsSuccess == false {
		doneChan <- false
		return
	}
	doneChan <- true
	return
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
