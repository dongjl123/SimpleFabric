package fabric

import (
	"os"
	"strconv"
)

type ProposalArgs struct {
}

type ProposalReply struct {
}

type PuArgs struct {
}
type PuReply struct {
}
type ReEvArgs struct {
}
type ReEvReply struct {
}
type OrderArgs struct {
}
type OrderReply struct {
}
type ReprArgs struct {
}
type ReprReply struct {
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the peer or orderer.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func peerSock() string {
	s := "/var/tmp/peer-"
	s += strconv.Itoa(os.Getuid())
	return s
}

func ordererSock() string {
	s := "/var/tmp/orderer-"
	s += strconv.Itoa(os.Getuid())
	return s
}
