package raft

import (
	"fmt"
	"sync"
	"time"
)

type NodeState int

const (
	Follower NodeState = iota
	Candidate
	Leader
)

var ErrNotLeader = fmt.Errorf("not leader")

var _ RaftNodeInterface = (*Node)(nil)

type Node struct {
	mu          sync.Mutex
	id          string
	state       NodeState
	currentTerm uint64
	votedFor    string

	log         []LogEntry
	commitIndex uint64
	lastApplied uint64

	fsm     *FSM
	applyCh chan ApplyResult
	stopCh  chan struct{}
}

type ApplyResult struct {
	Index   uint64
	Command []byte
	Result  any
}

func NewNode(id string, fsm *FSM) *Node {
	return &Node{
		id:      id,
		state:   Leader,
		fsm:     fsm,
		log:     make([]LogEntry, 0),
		applyCh: make(chan ApplyResult, 100),
		stopCh:  make(chan struct{}),
	}
}

func (n *Node) Start() {
	go n.runApplyLoop()
}

func (n *Node) Apply(cmd []byte) (uint64, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.state != Leader {
		return 0, ErrNotLeader
	}

	entry := LogEntry{
		Term:    n.currentTerm,
		Index:   uint64(len(n.log)) + 1,
		Command: cmd,
	}
	n.log = append(n.log, entry)
	n.commitIndex = entry.Index

	return entry.Index, nil
}

func (n *Node) runApplyLoop() {
	go func() {
		for {
			select {
			case <-n.stopCh:
				return
			default:
				n.mu.Lock()
				for n.lastApplied < n.commitIndex {
					entry := n.log[n.lastApplied]
					result := n.fsm.Apply(entry.Command)
					n.applyCh <- ApplyResult{
						Index:   entry.Index,
						Command: entry.Command,
						Result:  result,
					}
					n.lastApplied++
				}
				n.mu.Unlock()
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()
}

func (n *Node) Stop() {
	close(n.stopCh)
}

func (n *Node) IsLeader() bool {
	return n.state == Leader
}
