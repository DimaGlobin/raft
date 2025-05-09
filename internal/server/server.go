package server

import (
	"context"
	"net/http"

	"github.com/DimaGlobin/raft/internal/model/config"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	httpServer http.Server
}

func NewServer(cfg config.Config, router chi.Router) *Server {
	return &Server{
		httpServer: http.Server{
			Addr:    "localhost:" + cfg.Port,
			Handler: router,
		},
	}
}

func (s *Server) Run() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
