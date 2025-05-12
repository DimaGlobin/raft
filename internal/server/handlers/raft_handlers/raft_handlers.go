package raft_handlers

import (
	"encoding/json"
	"net/http"

	"github.com/DimaGlobin/raft/internal/service"
	"github.com/DimaGlobin/raft/pkg/raft"
	"golang.org/x/exp/slog"
)

type RaftHandler struct {
	srv service.Service
	log *slog.Logger
}

func NewRaftHandler(srv service.Service, log *slog.Logger) *RaftHandler {
	return &RaftHandler{
		srv: srv,
		log: log,
	}
}

func (rh *RaftHandler) Append(w http.ResponseWriter, r *http.Request) {
	rh.log.Debug("RaftHandler.Append called")

	var req raft.AppendEntriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	resp := rh.srv.Raft().HandleAppendEntries(req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (rh *RaftHandler) Vote(w http.ResponseWriter, r *http.Request) {
	var req raft.RequestVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	resp := rh.srv.Raft().HandleRequestVote(req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
