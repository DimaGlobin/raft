package router

import (
	"net/http"

	"github.com/DimaGlobin/raft/internal/middleware/logger"
	"github.com/DimaGlobin/raft/internal/server/handlers/kv_handlers"
	"github.com/DimaGlobin/raft/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/exp/slog"
)

func NewRouter(log *slog.Logger, service service.Service, self string) chi.Router {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	// router.Use(middleware.Logger)
	router.Use(logger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	kvHandler := kv_handlers.NewKvHandler(service, log, self)

	router.Route("/kv", func(r chi.Router) {
		r.Delete("/{key}", kvHandler.Delete)
		r.Get("/{key}", kvHandler.Get)
		r.Post("/", kvHandler.Create)
		r.Put("/{key}", kvHandler.Update)
		r.Patch("/{key}", kvHandler.Patch)
	})

	return router
}
