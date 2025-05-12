package raft

import "time"

type RaftNodeInterface interface {
	Start()
	Apply(cmd []byte) (uint64, error)
	IsLeader() bool
	HandleAppendEntries(req AppendEntriesRequest) AppendEntriesResponse
	HandleRequestVote(req RequestVoteRequest) RequestVoteResponse
	WaitForCommit(index uint64, timeout time.Duration) error
}
