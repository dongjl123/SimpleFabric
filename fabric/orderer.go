package fabric

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
)

var TxChan chan Transaction

type OrderBuf struct {
	mutx    sync.Mutex
	TxSlice []Transaction
}

type PrimaryPeerMap struct {
	mutx  sync.Mutex
	prMap map[string]string
}

//存储orderer信息的结构体，维护orderer自身的一些状态
type Orderer struct {
	organization string
	orderId      string
	orderArgsBuf *OrderBuf
	prPeerMap    *PrimaryPeerMap
	// blockLedger  LedgerManager
	maxBlockSize int
}

type Block struct {
	TxSlice []Transaction
}

func (o *Orderer) GenerateBlock() {
	for {
		select {
		case newTx := <-TxChan:
			o.orderArgsBuf.mutx.Lock()
			o.orderArgsBuf.TxSlice = append(o.orderArgsBuf.TxSlice, newTx)
			if len(o.orderArgsBuf.TxSlice) >= o.maxBlockSize {
				block := Block{TxSlice: o.orderArgsBuf.TxSlice[:o.maxBlockSize]}
				o.orderArgsBuf.TxSlice = o.orderArgsBuf.TxSlice[o.maxBlockSize:]
				//排序节点发送区块给各组织主节点
				for org, peerid := range o.prPeerMap.prMap {
					args := PuArgs{Block: block}
					reply := PuReply{}
					go call(org, peerid, "PushBlock", &args, &reply)
				}
			}
			o.orderArgsBuf.mutx.Unlock()
		}
	}
}

//客户端调用，发送交易给排序节点
func (o *Orderer) TransOrder(args *OrderArgs, reply *OrderReply) {
	TxChan <- args.TX
	reply.IsSuccess = true
	return
}

//peer调用，在排序节点上注册为主节点
func (o *Orderer) RegisterPrimary(args *ReprArgs, reply *ReprReply) {
	o.prPeerMap.mutx.Lock()
	o.prPeerMap.prMap[args.org] = args.peerid
	o.prPeerMap.mutx.Unlock()
	reply.IsSuccess = true
	return
}

func (o *Orderer) Server() {
	rpc.Register(o)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := peerSock(o.organization, o.orderId)
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	TxChan = make(chan Transaction, 10)
	go o.GenerateBlock()
	go http.Serve(l, nil)
}

func NewOrder(org string, orderid string, maxblockszie int) (*Orderer, error) {
	o := Orderer{organization: org, orderId: orderid, maxBlockSize: maxblockszie}
	o.orderArgsBuf = &OrderBuf{TxSlice: make([]Transaction, 2*o.maxBlockSize)}
	o.prPeerMap = &PrimaryPeerMap{prMap: make(map[string]string)}
	// currentDir, _ := os.Getwd()
	// ledgerPath := currentDir + "/" + org + "_" + orderid
	// o.blockLedger = LedgerManager{dir: ledgerPath, blockHeight: 0}
	return &o, nil
}
