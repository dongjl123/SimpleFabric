# SimpleFabric
原生fabric过于复杂，该项目模拟fabric主要工作逻辑，方便验证技术改进效果。



## 通信协议

RPC（暂时使用go net/rpc 库，默认使用gob编码）



## 节点类型

- client
- peer
- orderer



## RPC远程服务函数

Peer远程服务函数

```go
TransProposal(proposalArgs, proposalReply) //客户端调用，发送交易提案给Peer

PushBlock(puArgs, puReply) //排序节点调用，发送区块给主节点

RegisterEvent(reEvArgs, reEvArgs) //注册事件，由客户调用，监听自己的交易是否被成功commit
```



Orderer远程服务函数

```go
TransOrder(orderArgs, orderReply) //客户端调用，发送交易给排序节点

RegisterPrimary(reprArgs, reprReply) //peer调用，在排序节点上注册为主节点
```



## 本地函数

Client本地函数

- 生成交易提案
- 发送交易提案
- 生成交易
- 发送交易
- 监听交易事件



Peer本地函数

- 模拟执行
- 模拟读数据库
- 模拟写数据库
- 生成读写集
- 区块验证
- commit账本
- 更新数据库



Orderer本地函数（做单节点版本）

- 区块处理器
- 生成区块
- 发送区块



## 数据结构

```go
type TransProposal struct{
    
}

type Transaction struct{
    
}


```

