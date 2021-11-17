package fabric

type ProposalArgs struct {
	TP TransProposal
}

type ProposalReply struct {
	IsSuccess bool
	RW        RWSet
}

type PuArgs struct {
	Block
}
type PuReply struct {
	IsSuccess bool
}

// type ReEvArgs struct {
// 	TxID string
// }
// type ReEvReply struct {
// 	IsSuccess bool
// }
type GetValidateMapArgs struct {
}
type GetValidateMapReply struct {
	ValidateTxs map[string]bool
}

type OrderArgs struct {
	TX Transaction
}
type OrderReply struct {
	IsSuccess bool
}
type ReprArgs struct {
	Peerid string
	Org    string
}
type ReprReply struct {
	IsSuccess bool
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the peer or orderer.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func peerSock(org, peerid string) string {
	s := "/var/tmp/peer-"
	s = s + org + "-" + peerid
	return s
}

// func ordererSock(org, peerid string) string {
// 	s := "/var/tmp/orderer-"
// 	s = s + org + "-" + peerid
// 	return s
// }
