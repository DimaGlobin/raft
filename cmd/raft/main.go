package main

import (
	"os"

	"github.com/DimaGlobin/raft/internal/model/config"
	"github.com/DimaGlobin/raft/internal/repository"
	"github.com/DimaGlobin/raft/internal/server"
	"github.com/DimaGlobin/raft/internal/server/router"
	"github.com/DimaGlobin/raft/internal/service"
	"github.com/DimaGlobin/raft/pkg/raft"
	"golang.org/x/exp/slog"
)

func main() {
	cfg := config.Load()

	log := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
	)

	repository := repository.New()

	raftCfg := raft.NodeConfig{
		ID:     cfg.Id,
		Peers:  cfg.Peers,
		Logger: log,
	}

	fsm := raft.NewFSM(repository, log)
	raftNode := raft.NewNode(fsm, raftCfg)
	raftNode.Start()

	service := service.NewKvService(repository, raftNode, log)
	router := router.NewRouter(log, service, cfg.Self)

	srv := server.NewServer(cfg, router)
	if err := srv.Run(); err != nil {
		log.Error("Cannot run server", err.Error())
		return
	}

	log.Info("server started")
}
