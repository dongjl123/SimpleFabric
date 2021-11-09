package fabric

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
	"strconv"
	"time"
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
	FunName  string
	Args     [3]string
	Identity string
}

//交易，应该包含交易ID，交易的读写集，身份信息。
type Transaction struct {
	TxID     string
	Identity string
	RWSet
}

func NewTxProposal(FunName string, Args [3]string, Identity string) TransProposal {
	transProposal := TransProposal{FunName: FunName, Args: Args, Identity: Identity}
	return transProposal
}

func SendProposal(org string, peerID string, txp TransProposal) (ProposalReply, error) {
	sendArgs := ProposalArgs{TP: txp}
	sendReply := ProposalReply{}
	err := call(org, peerID, "Peer.TransProposal", &sendArgs, &sendReply)
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

func NewTx(rwset RWSet, Identity string) (Transaction, error) {
	txID, err := makeTxID(Identity, rwset)
	fmt.Println(txID)
	if err != nil {
		fmt.Println("'make TxID ERROR: ", err)
		return Transaction{}, err
	}
	newTx := Transaction{RWSet: rwset, Identity: Identity, TxID: txID}
	return newTx, nil
}

func SendTx(Tx Transaction) (OrderReply, error) {
	ordArgs := OrderArgs{TX: Tx}
	ordReply := OrderReply{}
	err := call("orderorg", "orderer1", "Orderer.TransOrder", &ordArgs, &ordReply)
	return ordReply, err
}

// func ClientRegisterEvent(txid string) (ReEvReply, error) {
// 	reArgs := ReEvArgs{TxID: txid}
// 	reReply := ReEvReply{}
// 	err := call(orgs[0], peers[0], "Peer.RegisterEvent", &reArgs, &reReply)
// 	return reReply, err
// }

func ClientRegisterEvent(txid string) *rpc.Call {
	reArgs := ReEvArgs{TxID: txid}
	reReply := ReEvReply{}
	return asyncCall(orgs[0], peers[0], "Peer.RegisterEvent", &reArgs, &reReply)

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
//因为在做本地通信测试时发现交易结束的太快
//以至于来不及注册监听而发送永久阻塞，故先注册监听再发送交易
func Client(id int, doneChan chan bool) {
	//生成交易提案，发送交易提案
	Identity := "client" + strconv.Itoa(id)
	Args := [3]string{"Alice", "Bob", "10"}
	txp := NewTxProposal("transfer", Args, Identity)
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
	Tx, err := NewTx(RWSlice[0], Identity)
	if err != nil {
		fmt.Println("New TX ERROR")
		doneChan <- false
		return
	}
	eventChan := ClientRegisterEvent(Tx.TxID).Done
	time.Sleep(time.Second)
	sendReply, err := SendTx(Tx)
	if err != nil || sendReply.IsSuccess == false {
		fmt.Println("send Tx fail:", err)
		doneChan <- false
		return
	}
	// fmt.Println("send TX finish")
	//监听
	select {
	case event := <-eventChan:
		reply := event.Reply.(*ReEvReply)
		err := event.Error
		if err != nil {
			fmt.Println("listen registered event error", err)
			doneChan <- false
			return
		}
		if reply.IsSuccess == false {
			// fmt.Println("TX fail")
			doneChan <- false
			return
		}
	}
	doneChan <- true
	return
	// reReply, err := ClientRegisterEvent(Tx.TxID)
	// if err != nil {
	// 	fmt.Println("listen registered event error")
	// 	doneChan <- false
	// 	return
	// }
	// if reReply.IsSuccess == false {
	// 	doneChan <- false
	// 	return
	// }
	// fmt.Println("event reply is right")
	// doneChan <- true
	// return
}

func asyncCall(org string, peerid string, rpcname string, Args interface{}, reply interface{}) *rpc.Call {
	//c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := peerSock(org, peerid)
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	return c.Go(rpcname, Args, reply, nil)
}

func call(org string, peerid string, rpcname string, Args interface{}, reply interface{}) error {
	//c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := peerSock(org, peerid)
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()
	err = c.Call(rpcname, Args, reply)
	return err
}
