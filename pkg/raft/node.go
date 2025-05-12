package raft

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"

	"golang.org/x/exp/slog"
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
	logger  *slog.Logger

	peers      map[string]string
	nextIndex  map[string]uint64
	matchIndex map[string]uint64

	lastHeartbeat time.Time
}

type ApplyResult struct {
	Index   uint64
	Command []byte
	Result  any
}

func NewNode(fsm *FSM, cfg NodeConfig) *Node {
	nextIndex := make(map[string]uint64)
	matchIndex := make(map[string]uint64)
	for id := range cfg.Peers {
		nextIndex[id] = 1
		matchIndex[id] = 0
	}

	return &Node{
		id:            cfg.ID,
		state:         Follower,
		fsm:           fsm,
		log:           make([]LogEntry, 0),
		applyCh:       make(chan ApplyResult, 100),
		stopCh:        make(chan struct{}),
		peers:         cfg.Peers,
		lastHeartbeat: time.Now(),
		logger:        cfg.Logger,
		matchIndex:    matchIndex,
		nextIndex:     nextIndex,
	}
}

func (n *Node) Start() {
	n.logger.Debug("Node start initiated")

	go n.runApplyLoop()
	go n.runHeartbeatLoop()
	go n.runElectionLoop()

	n.logger.Debug("All background loops started")
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

	// n.updateCommitIndex()

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

					n.logger.Debug("runApplyLoop applying", "index", entry.Index)

					result := n.fsm.Apply(entry.Command)

					n.logger.Debug("runApplyLoop applied", "index", entry.Index, "result", result)

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
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.state == Leader
}

func (n *Node) runHeartbeatLoop() {
	n.logger.Debug("runHeartbeatLoop sending to peers", "state", n.state)

	go func() {
		for {
			select {
			case <-n.stopCh:
				return
			default:
				n.mu.Lock()
				isLeader := n.state == Leader
				state := n.state
				n.mu.Unlock()

				n.logger.Debug("Heartbeat tick", "state", state, "isLeader", isLeader)

				if isLeader {
					n.logger.Debug("Heartbeat tick - sending AppendEntries")
					for peerID := range n.peers {
						go n.sendAppendEntries(peerID)
					}
				} else {
					n.logger.Debug("Heartbeat tick - not leader, skipping")
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()
}

func (n *Node) HandleAppendEntries(req AppendEntriesRequest) AppendEntriesResponse {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.logger.Info("Received AppendEntries",
		"from", req.LeaderID,
		"term", req.Term,
		"prevLogIndex", req.PrevLogIndex,
		"prevLogTerm", req.PrevLogTerm,
		"entries", len(req.Entries),
		"leaderCommit", req.LeaderCommit,
	)

	if req.Term > n.currentTerm {
		n.logger.Info("Updating term and stepping down to Follower", "fromTerm", n.currentTerm, "toTerm", req.Term)
		n.logger.Warn("Got AppendEntries with higher term, stepping down", "currentTerm", n.currentTerm, "newTerm", req.Term)
		n.currentTerm = req.Term
		n.state = Follower
		n.votedFor = ""
	}

	if req.Term < n.currentTerm {
		n.logger.Warn("Rejecting AppendEntries due to stale term", "currentTerm", n.currentTerm, "reqTerm", req.Term)
		return AppendEntriesResponse{
			Term:    n.currentTerm,
			Success: false,
		}
	}

	n.lastHeartbeat = time.Now()

	if req.PrevLogIndex > 0 {
		if int(req.PrevLogIndex) > len(n.log) {
			n.logger.Warn("Rejecting AppendEntries: PrevLogIndex out of bounds", "prevLogIndex", req.PrevLogIndex, "logLength", len(n.log))
			return AppendEntriesResponse{Term: n.currentTerm, Success: false}
		}
		prev := n.log[req.PrevLogIndex-1]
		if prev.Term != req.PrevLogTerm {
			n.logger.Warn("Rejecting AppendEntries: log term mismatch", "prevLogIndex", req.PrevLogIndex, "expectedTerm", prev.Term, "gotTerm", req.PrevLogTerm)
			return AppendEntriesResponse{Term: n.currentTerm, Success: false}
		}
	}

	if req.PrevLogIndex < uint64(len(n.log)) {
		n.log = n.log[:req.PrevLogIndex]
	}
	n.log = append(n.log, req.Entries...)
	n.logger.Info("Log updated", "newLength", len(n.log))

	if req.LeaderCommit > n.commitIndex {
		lastNew := req.PrevLogIndex + uint64(len(req.Entries))
		if req.LeaderCommit < lastNew {
			n.commitIndex = req.LeaderCommit
		} else {
			n.commitIndex = lastNew
		}
		n.logger.Debug("Commit index updated", "commitIndex", n.commitIndex)
	}

	return AppendEntriesResponse{
		Term:    n.currentTerm,
		Success: true,
	}
}

func (n *Node) sendAppendEntries(peerID string) {
	n.logger.Debug("sendAppendEntries start", "peer", peerID)

	n.mu.Lock()

	addr, ok := n.peers[peerID]
	if !ok {
		n.logger.Warn("Peer not found", "peer", peerID)
		n.mu.Unlock()
		return
	}

	nextIdx := n.nextIndex[peerID]

	var prevLogTerm uint64
	if nextIdx > 1 && int(nextIdx-1) <= len(n.log) {
		prevLogTerm = n.log[nextIdx-2].Term
	}

	entries := make([]LogEntry, 0)
	if int(nextIdx-1) < len(n.log) {
		entries = append(entries, n.log[nextIdx-1:]...)
	}

	req := AppendEntriesRequest{
		Term:         n.currentTerm,
		LeaderID:     n.id,
		PrevLogIndex: nextIdx - 1,
		PrevLogTerm:  prevLogTerm,
		Entries:      entries,
		LeaderCommit: n.commitIndex,
	}

	n.mu.Unlock()

	n.logger.Debug("Sending AppendEntries",
		"peer", peerID,
		"entries", len(entries),
		"nextIndex", nextIdx,
		"logLen", len(n.log))

	data, err := json.Marshal(req)
	if err != nil {
		n.logger.Error("failed to marshal AppendEntries request", "peer", peerID, "error", err)
		return
	}

	n.logger.Debug("sending HTTP request", "peer", peerID, "url", addr+"/raft/append")
	resp, err := http.Post(addr+"/raft/append", "application/json", bytes.NewReader(data))
	if err != nil {
		n.logger.Warn("AppendEntries failed", "peer", peerID, "error", err)
		return
	}
	defer resp.Body.Close()

	n.logger.Debug("AppendEntries sent", "peer", peerID, "status", resp.StatusCode)

	var res AppendEntriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		n.logger.Warn("Decode AppendEntriesResponse failed", "peer", peerID, "error", err)
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if res.Term > n.currentTerm {
		n.currentTerm = res.Term
		n.state = Follower
		n.votedFor = ""
		n.logger.Info("Stepping down due to higher term", "term", res.Term)
		return
	}

	if res.Success {
		n.matchIndex[peerID] = req.PrevLogIndex + uint64(len(req.Entries))
		n.nextIndex[peerID] = n.matchIndex[peerID] + 1
		n.updateCommitIndex()
		n.logger.Debug("AppendEntries success", "peer", peerID, "nextIndex", n.nextIndex[peerID])
	} else {
		if n.nextIndex[peerID] > 1 {
			n.nextIndex[peerID]--
			n.logger.Debug("AppendEntries failed, decreasing nextIndex", "peer", peerID, "newNext", n.nextIndex[peerID])
		}
	}
}

func (n *Node) HandleRequestVote(req RequestVoteRequest) RequestVoteResponse {
	n.mu.Lock()
	defer n.mu.Unlock()

	if req.Term > n.currentTerm {
		n.currentTerm = req.Term
		n.state = Follower
		n.votedFor = ""
	}

	if req.Term < n.currentTerm {
		return RequestVoteResponse{Term: n.currentTerm, VoteGranted: false}
	}

	if n.votedFor != "" && n.votedFor != req.CandidateID {
		return RequestVoteResponse{Term: n.currentTerm, VoteGranted: false}
	}

	lastIndex := uint64(len(n.log))
	lastTerm := uint64(0)
	if lastIndex > 0 {
		lastTerm = n.log[lastIndex-1].Term
	}

	if req.LastLogTerm < lastTerm || (req.LastLogTerm == lastTerm && req.LastLogIndex < lastIndex) {
		return RequestVoteResponse{Term: n.currentTerm, VoteGranted: false}
	}

	n.votedFor = req.CandidateID
	return RequestVoteResponse{Term: n.currentTerm, VoteGranted: true}
}

func (n *Node) sendRequestVote(peerID string) (RequestVoteResponse, error) {
	addr, ok := n.peers[peerID]
	if !ok {
		return RequestVoteResponse{}, fmt.Errorf("unknown peer: %s", peerID)
	}

	lastIndex := uint64(len(n.log))
	lastTerm := uint64(0)
	if lastIndex > 0 {
		lastTerm = n.log[lastIndex-1].Term
	}

	req := RequestVoteRequest{
		Term:         n.currentTerm,
		CandidateID:  n.id,
		LastLogIndex: lastIndex,
		LastLogTerm:  lastTerm,
	}

	data, _ := json.Marshal(req)
	resp, err := http.Post(addr+"/raft/vote", "application/json", bytes.NewReader(data))
	if err != nil {
		return RequestVoteResponse{}, err
	}
	defer resp.Body.Close()

	var res RequestVoteResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	return res, err
}

func (n *Node) runElectionLoop() {
	for {
		select {
		case <-n.stopCh:
			return
		default:
			time.Sleep(10 * time.Millisecond)

			n.mu.Lock()
			if n.state == Leader {
				n.mu.Unlock()
				continue
			}

			if time.Since(n.lastHeartbeat) >= randomElectionTimeout() {
				n.mu.Unlock()
				n.startElection()
			} else {
				n.mu.Unlock()
			}
		}
	}
}

func (n *Node) startElection() {
	n.logger.Debug("startElection triggered", "term", n.currentTerm+1)

	n.mu.Lock()
	n.currentTerm++
	n.state = Candidate
	n.votedFor = n.id
	term := n.currentTerm
	n.lastHeartbeat = time.Now()
	n.logger.Info("Election started", "term", term, "candidate", n.id)
	n.mu.Unlock()

	votes := 1
	var wg sync.WaitGroup
	var mu sync.Mutex

	for peerID := range n.peers {
		wg.Add(1)
		go func(pid string) {
			defer wg.Done()
			res, err := n.sendRequestVote(pid)
			if err != nil {
				n.logger.Error("Failed to send RequestVote", "to", pid, "error", err)
				return
			}
			if res.VoteGranted {
				n.logger.Debug("Vote granted", "from", pid)
				mu.Lock()
				votes++
				mu.Unlock()
			} else {
				n.logger.Warn("Vote denied", "from", pid, "term", res.Term)
			}
		}(peerID)
	}

	wg.Wait()

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.currentTerm == term && votes > len(n.peers)/2 {
		n.state = Leader
		n.votedFor = ""
		n.logger.Info("Node became leader", "id", n.id, "term", n.currentTerm, "votes", votes)
	} else {
		n.logger.Info("Election lost or stale term", "term", n.currentTerm, "votes", votes)
	}
}

func randomElectionTimeout() time.Duration {
	return time.Duration(500+rand.Intn(500)) * time.Millisecond
}

func (n *Node) updateCommitIndex() {
	n.logger.Debug("updateCommitIndex called", "matchIndex", n.matchIndex, "ownIndex", len(n.log))

	matchIndexes := make([]uint64, 0, len(n.matchIndex)+1)

	ownLastIndex := uint64(0)
	if len(n.log) > 0 {
		ownLastIndex = n.log[len(n.log)-1].Index
	}
	matchIndexes = append(matchIndexes, ownLastIndex)

	for _, idx := range n.matchIndex {
		matchIndexes = append(matchIndexes, idx)
	}

	sort.Slice(matchIndexes, func(i, j int) bool {
		return matchIndexes[i] > matchIndexes[j]
	})

	quorumIndex := matchIndexes[len(matchIndexes)/2]

	if quorumIndex <= n.commitIndex {
		return
	}

	if int(quorumIndex) <= len(n.log) && n.log[quorumIndex-1].Term == n.currentTerm {
		n.commitIndex = quorumIndex
		n.logger.Info("Updated commitIndex", "commitIndex", n.commitIndex)
	}
}

func (n *Node) WaitForCommit(index uint64, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			return fmt.Errorf("timeout waiting for commit of index %d", index)
		case <-ticker.C:
			n.mu.Lock()
			committed := n.commitIndex >= index
			n.mu.Unlock()
			if committed {
				return nil
			}
		}
	}
}

func (n *Node) ApplyWithResult(data []byte, timeout time.Duration) (any, error) {
	n.logger.Debug("ApplyWithResult called")

	index, err := n.Apply(data)
	if err != nil {
		return nil, err
	}

	n.logger.Debug("waiting for apply result", "index", index)

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return nil, fmt.Errorf("timeout waiting for apply result, index=%d", index)
		case res := <-n.applyCh:
			n.logger.Debug("applyCh received", "index", res.Index, "expectedIndex", index)
			if res.Index == index {
				return res.Result, nil
			}
		}
	}
}
