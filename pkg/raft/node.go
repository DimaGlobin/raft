package raft

type Node struct {
	isLeader bool
	fsm FSM
}

func (n Node) IsLeader() bool {
	return n.isLeader
}

func (n Node) Apply()