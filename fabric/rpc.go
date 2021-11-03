package fabric

type ProposalArgs struct {
	TP TransProposal
}

type ProposalReply struct {
	IsSuccess bool
	RW        RWSet
}

type PuArgs struct {
}
type PuReply struct {
}
type ReEvArgs struct {
	TxID string
}
type ReEvReply struct {
	IsSuccess bool
}
type OrderArgs struct {
	TX Transaction
}
type OrderReply struct {
	IsSuccess bool
}
type ReprArgs struct {
	peerid string
	org    string
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

func ordererSock(org, peerid string) string {
	s := "/var/tmp/orderer-"
	s = s + org + "-" + peerid
	return s
}
