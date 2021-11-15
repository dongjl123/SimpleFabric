package fabric

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"sync"
	"time"
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
	BlockHeight  int
	maxBlockSize int
}

type Block struct {
	BlockHeight int
	TxSlice     []Transaction
}

func (o *Orderer) cutBlock(generateType int) {
	var block Block
	if generateType == 1 {
		if len(o.orderArgsBuf.TxSlice) >= o.maxBlockSize {
			block = Block{BlockHeight: o.BlockHeight, TxSlice: o.orderArgsBuf.TxSlice[:o.maxBlockSize]}
			o.orderArgsBuf.TxSlice = o.orderArgsBuf.TxSlice[o.maxBlockSize:]
		} else {
			return
		}
	} else {
		if len(o.orderArgsBuf.TxSlice) > 0 {
			block = Block{BlockHeight: o.BlockHeight, TxSlice: o.orderArgsBuf.TxSlice}
			o.orderArgsBuf.TxSlice = o.orderArgsBuf.TxSlice[len(o.orderArgsBuf.TxSlice):]
		} else {
			return
		}
	}
	fmt.Println("Orderer generator block", o.BlockHeight, "and by type", generateType)
	o.BlockHeight++
	//排序节点发送区块给各组织主节点
	o.prPeerMap.mutx.Lock()
	for org, peerid := range o.prPeerMap.prMap {
		args := PuArgs{Block: block}
		reply := PuReply{}
		go call(org, peerid, "Peer.PushBlock", &args, &reply)
	}
	o.prPeerMap.mutx.Unlock()
}

func (o *Orderer) receiveTrans() {
	for {
		select {
		case newTx := <-TxChan:
			o.orderArgsBuf.mutx.Lock()
			o.orderArgsBuf.TxSlice = append(o.orderArgsBuf.TxSlice, newTx)
			o.cutBlock(1)
			o.orderArgsBuf.mutx.Unlock()
		}
	}
}

func (o *Orderer) generateBlockByTime() {
	tick := time.Tick(10e9)

	for {
		<-tick
		o.orderArgsBuf.mutx.Lock()
		o.cutBlock(0)
		o.orderArgsBuf.mutx.Unlock()
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
	go o.receiveTrans()
	go o.generateBlockByTime()
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
