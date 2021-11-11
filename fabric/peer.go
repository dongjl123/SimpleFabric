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

type KeyVersion struct {
	BlockHeight string
	TxID        string
}
type stateDBItem struct {
	Value   int
	version KeyVersion
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
	blockBuf     map[int]Block
}

type ReadItem struct {
	Key     string
	Value   int
	Version KeyVersion
}

type WriteItem struct {
	Key   string
	Value int
}
type RWSet struct {
	ReadSet  []ReadItem
	WriteSet []WriteItem
}

type ValidateTrascation struct {
	Transaction
	IsSuccess bool
}

//存储持有验证结果的交易的块
type ValidateBlock []ValidateTrascation

var blockChan chan Block

func (s *stateDB) getVersion(Key string) KeyVersion {
	s.RLock()
	defer s.RUnlock()
	return s.db[Key].version
}

func (s *stateDB) get(Key string) (int, KeyVersion) {
	s.RLock()
	defer s.RUnlock()
	return s.db[Key].Value, s.db[Key].version
}

func (s *stateDB) put(Key string, val int, ver KeyVersion) {
	s.Lock()
	defer s.Unlock()
	s.db[Key] = stateDBItem{Value: val, version: ver}
}

//链码函数，创建账户
func (p *Peer) createAccount(args [3]string) (RWSet, bool) {
	if _, ok := p.db.db[args[0]]; ok {
		fmt.Println("The account already exists!")
		return RWSet{}, false
	}
	depositNum, _ := strconv.Atoi(args[1])
	readKey1 := ReadItem{Key: args[0]}
	writeKey1 := WriteItem{Key: args[0], Value: depositNum}
	return RWSet{ReadSet: []ReadItem{readKey1}, WriteSet: []WriteItem{writeKey1}}, true
}

//链码函数，转账功能
func (p *Peer) transfer(args [3]string) (RWSet, bool) {
	val1, ver1 := p.db.get(args[0])
	val2, ver2 := p.db.get(args[1])
	transNum, _ := strconv.Atoi(args[2])
	//if val1 < transNum {
	//	return RWSet{}, false
	//}
	readKey1 := ReadItem{Key: args[0], Value: val1, Version: ver1}
	readKey2 := ReadItem{Key: args[1], Value: val2, Version: ver2}
	writeKey1 := WriteItem{Key: args[0], Value: val1 - transNum}
	writeKey2 := WriteItem{Key: args[1], Value: val2 + transNum}
	return RWSet{ReadSet: []ReadItem{readKey1, readKey2}, WriteSet: []WriteItem{writeKey1, writeKey2}}, true
}

func (p *Peer) bePrimaryPeer() (ReprReply, error) {
	reprArgs := ReprArgs{Peerid: p.peerId, Org: p.organization}
	reprReply := ReprReply{}
	err := call("orderorg", "orderer1", "Orderer.RegisterPrimary", &reprArgs, &reprReply)
	return reprReply, err
}

func (p *Peer) pubilshEvent(b ValidateBlock) {
	for _, i := range b {
		p.eventList.RLock()
		// fmt.Println("read lock right")
		e, ok := p.eventList.informChans[i.Transaction.TxID]
		p.eventList.RUnlock()
		// fmt.Println("read unlock right")
		if !ok {
			continue
		}
		if i.IsSuccess {
			e <- true
		} else {
			e <- false
		}
	}
	return
}

//验证函数，比较读键的版本
func (p *Peer) validate(b Block) ValidateBlock {
	vb := ValidateBlock{}
	outOfDateKeySet := make(map[string]bool) //在当前块中验证成功的交易的写键集合
	for _, tx := range b.TxSlice {
		isRight := true
		//比较读键
		for _, ri := range tx.RWSet.ReadSet {
			if _, ok := outOfDateKeySet[ri.Key]; ok {
				isRight = false
				fmt.Println(tx, outOfDateKeySet, "conflict in the block")
				break
			}
			if ri.Version != p.db.getVersion(ri.Key) {
				isRight = false
				fmt.Println("ri:", ri, "get from db:", p.db.getVersion(ri.Key))
				fmt.Println("conflict because of previous block")
				break
			}
		}
		if isRight {
			for _, wi := range tx.RWSet.WriteSet {
				outOfDateKeySet[wi.Key] = true
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
	// p.blockLedger.blockHeight += 1
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
				p.db.put(wk.Key, wk.Value, KeyVersion{BlockHeight: strconv.Itoa(p.blockLedger.blockHeight), TxID: vtx.TxID})
			}
		}
	}
}

func (p *Peer) handleBlock() {
	for {
		select {
		case newBlock := <-blockChan:
			// 将收到的块放入buffer中
			p.blockBuf[newBlock.BlockHeight] = newBlock
			fmt.Println(newBlock.BlockHeight, p.blockLedger.blockHeight)
			// 根据blockLedger.blockHeight从buffer中读取下一个块
			for {
				if _, ok := p.blockBuf[p.blockLedger.blockHeight+1]; !ok {
					break
				}
				p.blockLedger.blockHeight += 1
				nextBlock := p.blockBuf[p.blockLedger.blockHeight]
				validateNewBlock := p.validate(nextBlock)
				p.commiter(nextBlock)
				p.updateDB(validateNewBlock)
				p.pubilshEvent(validateNewBlock)
				delete(p.blockBuf, p.blockLedger.blockHeight)
			}

		}
	}
}

//客户端调用，发送交易提案给Peer
func (p *Peer) TransProposal(args *ProposalArgs, reply *ProposalReply) error {
	if args.TP.FunName == "transfer" {
		reply.RW, reply.IsSuccess = p.transfer(args.TP.Args)
	} //后续在这里可以用if else添加其他处理函数
	if args.TP.FunName == "createAccount" {
		reply.RW, reply.IsSuccess = p.createAccount(args.TP.Args)
	}
	fmt.Println(reply.RW)
	return nil
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
func (p *Peer) PushBlock(puArgs *PuArgs, puReply *PuReply) error {
	blockChan <- puArgs.Block
	//如果是主节点，需要把区块同步推送到组织内其他节点
	if p.isPrPeer == true {
		var wg sync.WaitGroup
		wg.Add(len(peers) - 1)
		for _, otherPeer := range peers {
			if otherPeer != p.peerId {
				args := PuArgs{Block: puArgs.Block}
				reply := PuReply{}
				go callWait(&wg, p.organization, otherPeer, "Peer.PushBlock", &args, &reply)
			}
		}
		wg.Wait()
	}
	return nil
}

//注册事件，由客户调用，监听自己的交易是否被成功commit
func (p *Peer) RegisterEvent(reArgs *ReEvArgs, reReply *ReEvReply) error {
	//一个潜在的bug，如果交易ID发生了哈希碰撞，可能会导致RPC连接一直挂着
	// fmt.Println(reArgs.TxID)
	informChan := make(chan bool)
	p.eventList.Lock()
	// fmt.Println("write lock right")
	p.eventList.informChans[reArgs.TxID] = informChan
	p.eventList.Unlock()
	// fmt.Println("write unlock right")
	select {
	case isSuccess := <-informChan:
		// fmt.Println("get inform")
		if isSuccess == true {
			reReply.IsSuccess = true
		} else {
			reReply.IsSuccess = false
		}
	}
	return nil
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
	p.db.put("Bob", 1000, KeyVersion{BlockHeight: "0", TxID: "123"})
	p.db.put("Alice", 1000, KeyVersion{BlockHeight: "0", TxID: "123"})
	p.eventList = &eventHandler{informChans: make(map[string]chan bool)}
	p.blockBuf = make(map[int]Block)
	currentDir, _ := os.Getwd()
	ledgerPath := currentDir + "/" + org + "_" + peerid
	p.blockLedger = &LedgerManager{dir: ledgerPath, blockHeight: 0}
	if p.isPrPeer {
		reprReply, err := p.bePrimaryPeer()
		if err != nil || reprReply.IsSuccess == false {
			fmt.Println("register primary peer fail:", err)
			return nil, err
		}
	}
	return &p, nil
}
