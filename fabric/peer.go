package fabric

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

//一个块一个文件
type LedgerManager struct {
	dir         string
	blockHeight int
}

//存储peer信息的结构体，维护peer自身的一些状态
type Peer struct {
	organization string
	peerId       string
	stateDB      map[string]string
	blockLedger  LedgerManager
}

type ReadItem struct {
	key     string
	value   string
	version string
}

type WriteItem struct {
	key   string
	value string
}
type RWSet struct {
	ReadSet  []ReadItem
	WriteSet []WriteItem
}

//客户端调用，发送交易提案给Peer
func (p *Peer) TransProposal(args *ProposalArgs, reply *ProposalReply) {

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
	sockname := peerSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

func NewPeer(org string, peerid string) (*Peer, error) {

}
