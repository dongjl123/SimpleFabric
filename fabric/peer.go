package fabric

//一个块一个文件
type LedgerManager struct {
	dir         string
	blockHeight int
}

//存储peer信息的结构体，维护peer自身的一些状态
type Peer struct {
	organization string
	peerId       int
	stateDB      map[string]string
	blockLedger  LedgerManager
}

//客户端调用，发送交易提案给Peer
func (p *Peer) TransProposal(args *ProposalArgs, reply *ProposalReply) {

}

//排序节点调用，发送区块给主节点
func PushBlock(PuArgs, PuReply) {

}

//注册事件，由客户调用，监听自己的交易是否被成功commit
func RegisterEvent(ReEvArgs, ReEvReply) {

}

func (p *Peer) Server() {

}

func NewPeer(org string, peerid int) *Peer {

}
