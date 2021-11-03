package fabric

//存储orderer信息的结构体，维护orderer自身的一些状态
type Orderer struct {
	orderId      string
	OrderArgsBuf []Transaction
	PrPeerMap    map[string]string
	blockLedger  LedgerManager
}

//客户端调用，发送交易给排序节点
func (o *Orderer) TransOrder(args *OrderArgs, reply *OrderReply) {
	o.OrderArgsBuf = append(o.OrderArgsBuf, args.TX)
	reply.IsSuccess = true
	return
}

//peer调用，在排序节点上注册为主节点
func (o *Orderer) RegisterPrimary(args *ReprArgs, reply *ReprReply) {
	o.PrPeerMap[args.org] = args.peerid
	reply.IsSuccess = true
	return
}
