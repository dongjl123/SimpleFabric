package fabric

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
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

type event

//存储peer信息的结构体，维护peer自身的一些状态
type Peer struct {
	organization string
	peerId       string
	db           stateDB
	blockLedger  LedgerManager
	eventList    
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

//客户端调用，发送交易提案给Peer
func (p *Peer) TransProposal(args *ProposalArgs, reply *ProposalReply) {
	if args.TP.funName == "transfer" {
		reply.RW = p.transfer(args.TP.args)
	} //后续在这里可以用if else添加其他处理函数
	reply.IsSuccess = true
	return
}

//排序节点调用，发送区块给主节点
func (p *Peer) PushBlock(PuArgs, PuReply) {

}

//注册事件，由客户调用，监听自己的交易是否被成功commit
func (p *Peer) RegisterEvent(ReEvArgs, ReEvReply) {

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
	go http.Serve(l, nil)
}

func NewPeer(org string, peerid string) (*Peer, error) {
	p := Peer{organization: org, peerId: peerid}
	p.db = make(stateDB)
	currentDir, _ := os.Getwd()
	ledgerPath := currentDir + "/" + org + "_" + peerid
	p.blockLedger = LedgerManager{dir: ledgerPath, blockHeight: 0}
	return &p, nil
}
