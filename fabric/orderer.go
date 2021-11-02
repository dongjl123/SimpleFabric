package fabric

//存储orderer信息的结构体，维护orderer自身的一些状态
type Orderer struct {
}

//客户端调用，发送交易给排序节点
func (o *Orderer) TransOrder(OrderArgs, OrderReply) {

}

//peer调用，在排序节点上注册为主节点
func (o *Orderer) RegisterPrimary(ReprArgs, ReprReply) {

}
