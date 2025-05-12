package raft

import (
	"golang.org/x/exp/slog"
)

type NodeConfig struct {
	ID     string
	Peers  map[string]string
	Logger *slog.Logger
}
