package fabric

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

//存储orderer信息的结构体，维护orderer自身的一些状态
type Orderer struct {
	organization string
	orderId      string
	orderArgsBuf []Transaction
	prPeerMap    map[string]string
	blockLedger  LedgerManager
	maxBlockSize int
}

type Block struct {
	TxSlice []Transaction
}

//客户端调用，发送交易给排序节点
func (o *Orderer) TransOrder(args *OrderArgs, reply *OrderReply) {
	o.orderArgsBuf = append(o.orderArgsBuf, args.TX)
	if len(o.orderArgsBuf) >= o.maxBlockSize {
		block := Block{TxSlice: o.orderArgsBuf[:o.maxBlockSize]}
		o.orderArgsBuf = o.orderArgsBuf[o.maxBlockSize:]
		//排序节点发送区块给各组织主节点
		for org, peerid := range o.prPeerMap {
			args := PuArgs{Block: block}
			reply := PuReply{}
			go call(org, peerid, "PushBlock", &args, &reply)
		}
	}
	reply.IsSuccess = true
	return
}

//peer调用，在排序节点上注册为主节点
func (o *Orderer) RegisterPrimary(args *ReprArgs, reply *ReprReply) {
	o.prPeerMap[args.org] = args.peerid
	reply.IsSuccess = true
	return
}

func (o *Orderer) Server() {
	rpc.Register(o)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	//这里是否需要修改为orderSock
	sockname := peerSock(o.organization, o.orderId)
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

func NewOrder(org string, orderid string, maxblockszie int) (*Orderer, error) {
	o := Orderer{organization: org, orderId: orderid, maxBlockSize: maxblockszie}
	o.orderArgsBuf = make([]Transaction, 2*o.maxBlockSize)
	o.prPeerMap = make(map[string]string)
	currentDir, _ := os.Getwd()
	ledgerPath := currentDir + "/" + org + "_" + orderid
	o.blockLedger = LedgerManager{dir: ledgerPath, blockHeight: 0}
	return &o, nil
}
