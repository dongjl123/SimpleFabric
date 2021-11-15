package fabric

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"sync"
)

var TxChan chan Transaction

type OrderBuf struct {
	// mutx    sync.Mutex
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
	BlockHeight  int
	maxBlockSize int
}

type Block struct {
	BlockHeight int
	TxSlice     []Transaction
}

func (o *Orderer) generateBlock() {
	for {
		select {
		case newTx := <-TxChan:
			o.orderArgsBuf.TxSlice = append(o.orderArgsBuf.TxSlice, newTx)
			if len(o.orderArgsBuf.TxSlice) >= o.maxBlockSize {
				block := Block{BlockHeight: o.BlockHeight, TxSlice: o.orderArgsBuf.TxSlice[:o.maxBlockSize]}
				o.BlockHeight++
				o.orderArgsBuf.TxSlice = o.orderArgsBuf.TxSlice[o.maxBlockSize:]
				//排序节点发送区块给各组织主节点
				o.prPeerMap.mutx.Lock()
				for org, peerid := range o.prPeerMap.prMap {
					args := PuArgs{Block: block}
					reply := PuReply{}
					go call(org, peerid, "Peer.PushBlock", &args, &reply)
				}
				o.prPeerMap.mutx.Unlock()
			}
		}
	}
}

//客户端调用，发送交易给排序节点
func (o *Orderer) TransOrder(args *OrderArgs, reply *OrderReply) error {
	TxChan <- args.TX
	reply.IsSuccess = true
	return nil
}

//peer调用，在排序节点上注册为主节点
func (o *Orderer) RegisterPrimary(args *ReprArgs, reply *ReprReply) error {
	o.prPeerMap.mutx.Lock()
	o.prPeerMap.prMap[args.Org] = args.Peerid
	o.prPeerMap.mutx.Unlock()
	reply.IsSuccess = true
	return nil
}

func (o *Orderer) Server() error {
	rpc.Register(o)
	rpc.HandleHTTP()
	address, err := getAddress(o.organization, o.orderId)
	if err != nil {
		fmt.Println("Server start error: ", err)
	}
	l, e := net.Listen("tcp", address)
	//sockname := peerSock(o.organization, o.orderId)
	//os.Remove(sockname)
	//l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	TxChan = make(chan Transaction, 1000)
	go o.generateBlock()
	go http.Serve(l, nil)
	return nil
}

func NewOrder(org string, orderid string, maxblockszie int) (*Orderer, error) {
	o := Orderer{organization: org, orderId: orderid, maxBlockSize: maxblockszie}
	o.orderArgsBuf = &OrderBuf{TxSlice: make([]Transaction, 0, 2*o.maxBlockSize)}
	o.prPeerMap = &PrimaryPeerMap{prMap: make(map[string]string)}
	o.BlockHeight = 1
	// currentDir, _ := os.Getwd()
	// ledgerPath := currentDir + "/" + org + "_" + orderid
	// o.blockLedger = LedgerManager{dir: ledgerPath, blockHeight: 0}
	return &o, nil
}
