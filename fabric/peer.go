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
	blockHeight string
	TxID        string
}
type stateDBItem struct {
	value   int
	version keyVersion
}

type stateDB struct {
	db map[string]stateDBItem
	sync.RWMutex
}

type eventHandler struct {
	informChans map[string]chan bool
	sync.RWMutex
}

//存储peer信息的结构体，维护peer自身的一些状态
type Peer struct {
	organization string
	peerId       string
	db           *stateDB
	blockLedger  *LedgerManager
	isPrPeer     bool
	eventList    *eventHandler
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

func (s *stateDB) getVersion(key string) keyVersion {
	s.RLock()
	defer s.RUnlock()
	return s.db[key].version
}

func (s *stateDB) get(key string) (int, keyVersion) {
	s.RLock()
	defer s.RUnlock()
	return s.db[key].value, s.db[key].version
}

func (s *stateDB) put(key string, val int, ver keyVersion) {
	s.Lock()
	defer s.Unlock()
	s.db[key] = stateDBItem{value: val, version: ver}
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

func (p *Peer) pubilshEvent(b ValidateBlock) {
	for _, i := range b {
		p.eventList.RLock()
		e, ok := p.eventList.informChans[i.Transaction.TxID]
		p.eventList.RUnlock()
		if !ok {
			continue
		}
		if i.IsSuccess {
			e <- true
		} else {
			e <- false
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

// create file with dir if dir is not exist
// path is dir
// name is file name
func createFileWithDir(path string, name string, content []byte) {
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		fmt.Println("Create block ledger file error", err)
	}
	file, err := os.OpenFile(path+"/"+name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Println("Open block ledger file error", err)
	}
	defer file.Close()
	_, err = file.Write(content)
	if err != nil {
		fmt.Println("Write block ledger file error", err)
	}
}

//这里写入账本的只是原始的区块，不带验证数据。只是一个模拟写账本文件的过程。
func (p *Peer) commiter(b Block) {
	fileName := "Block_" + strconv.Itoa(p.blockLedger.blockHeight)
	p.blockLedger.blockHeight += 1
	data, err := Encode(b)
	if err != nil {
		fmt.Println("encode block for commit error:", err)
	}
	createFileWithDir(p.blockLedger.dir, fileName, data)
}

//更新账本
func (p *Peer) updateDB(v ValidateBlock) {
	for _, vtx := range v {
		if vtx.IsSuccess {
			for _, wk := range vtx.Transaction.WriteSet {
				p.db.put(wk.key, wk.value, keyVersion{blockHeight: strconv.Itoa(p.blockLedger.blockHeight), TxID: vtx.TxID})
			}
		}
	}
}

func (p *Peer) handleBlock() {
	for {
		select {
		case newBlock := <-blockChan:
			validateNewBlock := p.validate(newBlock)
			p.commiter(newBlock)
			p.updateDB(validateNewBlock)
			p.pubilshEvent(validateNewBlock)
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
	informChan := make(chan bool)
	p.eventList.Lock()
	p.eventList.informChans[reArgs.TxID] = informChan
	p.eventList.Unlock()
	select {
	case isSuccess := <-informChan:
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
	p.db = &stateDB{db: make(map[string]stateDBItem)}
	p.eventList = &eventHandler{informChans: make(map[string]chan bool)}
	currentDir, _ := os.Getwd()
	ledgerPath := currentDir + "/" + org + "_" + peerid
	p.blockLedger = &LedgerManager{dir: ledgerPath, blockHeight: 0}
	if p.isPrPeer {
		reprReply, err := p.BePrimaryPeer()
		if err != nil || reprReply.IsSuccess == false {
			fmt.Println("register primary peer fail:", err)
			return nil, err
		}
	}
	return &p, nil
}
