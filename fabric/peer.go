package fabric

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"sync"
)

//一个块一个文件
type LedgerManager struct {
	dir         string
	blockHeight int
}

type keyVersion struct {
	blockHeight int
	TxID        int
}
type stateDBItem struct {
	value   int
	version keyVersion
}

type stateDB map[string]stateDBItem

type eventHandler struct {
	informChan chan bool
}

//存储peer信息的结构体，维护peer自身的一些状态
type Peer struct {
	organization string
	peerId       string
	db           stateDB
	blockLedger  LedgerManager
	isPrPeer     bool
	eventList    map[string]eventHandler
}

type ReadItem struct {
	key     string
	value   int
	version keyVersion
}

type WriteItem struct {
	key   string
	value int
}
type RWSet struct {
	ReadSet  []ReadItem
	WriteSet []WriteItem
}

type ValidateTrascation struct {
	Transaction
	IsSuccess bool
}

type ValidateBlock []ValidateTrascation

var blockChan chan Block

func (s stateDB) getVersion(key string) keyVersion {
	return s[key].version
}

func (s stateDB) get(key string) (int, keyVersion) {
	return s[key].value, s[key].version
}

func (s stateDB) put(key string, val int, ver keyVersion) {
	s[key] = stateDBItem{value: val, version: ver}
}

//链码函数，转账功能
func (p *Peer) transfer(args [3]string) RWSet {
	val1, ver1 := p.db.get(args[0])
	val2, ver2 := p.db.get(args[1])
	transNum, _ := strconv.Atoi(args[2])
	readKey1 := ReadItem{key: args[0], value: val1, version: ver1}
	readKey2 := ReadItem{key: args[1], value: val2, version: ver2}
	writeKey1 := WriteItem{key: args[0], value: val1 - transNum}
	writeKey2 := WriteItem{key: args[1], value: val2 + transNum}
	return RWSet{ReadSet: []ReadItem{readKey1, readKey2}, WriteSet: []WriteItem{writeKey1, writeKey2}}
}

func (p *Peer) BePrimaryPeer() (ReprReply, error) {
	reprArgs := ReprArgs{peerid: p.peerId, org: p.organization}
	reprReply := ReprReply{}
	err := call("orderorg", "orderer1", "RegisterPrimary", &reprArgs, &reprReply)
	return reprReply, err
}

func pubilshEvent(b ValidateBlock, eventList map[string]eventHandler) {
	for _, i := range b {
		e, ok := eventList[i.Transaction.TxID]
		if !ok {
			continue
		}
		if i.IsSuccess {
			e.informChan <- true
		} else {
			e.informChan <- false
		}
		return
	}
}

//验证函数，比较读键的版本
func (p *Peer) validate(b Block) ValidateBlock {
	vb := ValidateBlock{}
	for _, tx := range b.TxSlice {
		isRight := true
		//比较读键
		for _, ri := range tx.RWSet.ReadSet {
			if ri.version != p.db.getVersion(ri.key) {
				isRight = false
				break
			}
		}
		vb = append(vb, ValidateTrascation{Transaction: tx, IsSuccess: isRight})
	}
	return vb
}

//这里写入账本的只是原始的区块，不带验证数据。只是一个模拟写账本文件的过程。
func (p *Peer) commiter(b Block) {
	data, err := Encode(b)
	if err != nil {
		fmt.Println("encode block for commit error:", err)
		p.blockLedger.blockHeight += 1
	}

}

//更新账本，键版本号为"区块号#交易号"
func (p *Peer) updateDB(v ValidateBlock) {

}

func (p *Peer) handleBlock() {
	for {
		select {
		case newBlock := <-blockChan:
			validateNewBlock := p.validate(newBlock)
			p.commiter(newBlock)
			p.updateDB(validateNewBlock)
			pubilshEvent(validateNewBlock, p.eventList)
		}
	}
}

//客户端调用，发送交易提案给Peer
func (p *Peer) TransProposal(args *ProposalArgs, reply *ProposalReply) {
	if args.TP.funName == "transfer" {
		reply.RW = p.transfer(args.TP.args)
	} //后续在这里可以用if else添加其他处理函数
	reply.IsSuccess = true
	return
}

func callWait(wg *sync.WaitGroup, org string, peerid string, rpcname string, args interface{}, reply interface{}) {
	defer wg.Done()
	err := call(org, peerid, rpcname, args, reply)
	if err != nil {
		fmt.Println("send block error in organization:", err)
	}
	return
}

//排序节点调用，发送区块给主节点
func (p *Peer) PushBlock(puArgs PuArgs, puReply PuReply) {
	blockChan <- puArgs.Block
	//如果是主节点，需要把区块同步推送到组织内其他节点
	if p.isPrPeer == true {
		var wg sync.WaitGroup
		wg.Add(len(peers) - 1)
		for _, otherPeer := range peers {
			if otherPeer != p.peerId {
				args := PuArgs{Block: puArgs.Block}
				reply := PuReply{}
				go callWait(&wg, p.organization, otherPeer, "PushBlock", &args, &reply)
			}
		}
		wg.Wait()
	}
	return
}

//注册事件，由客户调用，监听自己的交易是否被成功commit
func (p *Peer) RegisterEvent(reArgs ReEvArgs, reReply ReEvReply) {
	//一个潜在的bug，如果交易ID发生了哈希碰撞，可能会导致RPC连接一直挂着
	newHandler := eventHandler{}
	newHandler.informChan = make(chan bool)
	p.eventList[reArgs.TxID] = newHandler
	select {
	case isSuccess := <-newHandler.informChan:
		if isSuccess == true {
			reReply.IsSuccess = true
		} else {
			reReply.IsSuccess = false
		}
	}
	return
}

func (p *Peer) Server() {
	rpc.Register(p)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := peerSock(p.organization, p.peerId)
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	blockChan = make(chan Block, 10)
	go p.handleBlock()
	go http.Serve(l, nil)
}

func NewPeer(org string, peerid string, isprpeer bool) (*Peer, error) {
	p := Peer{organization: org, peerId: peerid, isPrPeer: isprpeer}
	p.db = make(stateDB)
	p.eventList = make(map[string]eventHandler)
	currentDir, _ := os.Getwd()
	ledgerPath := currentDir + "/" + org + "_" + peerid
	p.blockLedger = LedgerManager{dir: ledgerPath, blockHeight: 0}
	if p.isPrPeer {
		reprReply, err := p.BePrimaryPeer()
		if err != nil || reprReply.IsSuccess == false {
			fmt.Println("register primary peer fail:", err)
			return nil, err
		}
	}
	return &p, nil
}
