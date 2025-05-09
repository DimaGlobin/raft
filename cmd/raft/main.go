package main

import (
	"os"

	"github.com/DimaGlobin/raft/internal/model/config"
	"github.com/DimaGlobin/raft/internal/repository"
	"github.com/DimaGlobin/raft/internal/server"
	"github.com/DimaGlobin/raft/internal/server/router"
	"github.com/DimaGlobin/raft/internal/service"
	"golang.org/x/exp/slog"
)

func main() {
	// cfg := config.Load()
	cfg := config.Config{
		IsLeader: true,
		Port:     "8080",
	}

	log := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
	)

	repository := repository.New()
	service := service.New(cfg, repository)
	router := router.NewRouter(log, service, cfg.Self)

	srv := server.NewServer(cfg, router)
	if err := srv.Run(); err != nil {
		log.Error("Cannot run server", err.Error())
		return
	}

	log.Info("server started")
}
