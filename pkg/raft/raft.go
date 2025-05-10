package raft

type RaftNodeInterface interface {
	Start()
	Apply(cmd []byte) (uint64, error)
	IsLeader() bool
}
