package raft

type LogEntry struct {
	Term    uint64
	Index   uint64
	Command []byte
}
