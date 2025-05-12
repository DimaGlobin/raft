package kv_handlers

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strings"

	"github.com/DimaGlobin/raft/internal/service"
	"github.com/go-chi/chi/v5"
	"golang.org/x/exp/slog"
)

type KvHandler struct {
	srv  service.Service
	log  *slog.Logger
	self string
}

func NewKvHandler(service service.Service, logger *slog.Logger, self string) *KvHandler { //TODO:refactor
	return &KvHandler{
		srv:  service,
		log:  logger,
		self: self,
	}
}

type kvPayload struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (k *KvHandler) redirectToReplica(w http.ResponseWriter, r *http.Request) bool {
	if k.srv.Raft().IsLeader() && (r.Method == http.MethodGet) && len(k.srv.GetReplicas()) != 0 {
		// pick random other
		replicas := k.srv.GetReplicas()
		reps := make([]string, 0, len(replicas)-1)
		for _, u := range replicas {
			if u != "http://"+k.self {
				reps = append(reps, u)
			}
		}
		tgt := reps[rand.Intn(len(reps))]
		w.Header().Set("Location", tgt+"/kv/"+chi.URLParam(r, "key"))
		w.WriteHeader(http.StatusFound)
		return true
	}
	return false
}

func (h *KvHandler) Create(w http.ResponseWriter, r *http.Request) {
	var payload kvPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if payload.Key == "" || payload.Value == "" {
		http.Error(w, "key and value required", http.StatusBadRequest)
		return
	}

	err := h.srv.Create(payload.Key, payload.Value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (k *KvHandler) Get(w http.ResponseWriter, r *http.Request) {
	if k.redirectToReplica(w, r) {
		return
	}

	key := chi.URLParam(r, "key")
	v, err := k.srv.Get(key)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Write([]byte(v))
}

func (h *KvHandler) Update(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	var payload kvPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if payload.Value == "" {
		http.Error(w, "value required", http.StatusBadRequest)
		return
	}

	err := h.srv.Update(key, payload.Value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *KvHandler) Patch(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	var payload map[string]string
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	value, ok := payload["value"]
	if !ok {
		http.Error(w, "value field required", http.StatusBadRequest)
		return
	}

	err := h.srv.Patch(key, value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *KvHandler) Delete(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	err := h.srv.Delete(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (k *KvHandler) CompareAndSwap(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	// Читаем новое значение из тела запроса
	var body struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	expected := r.Header.Get("If-Match")

	if r.Header.Get("If-None-Match") == "*" {
		expected = ""
	}

	err := k.srv.CAS(key, expected, body.Value)
	if err != nil {
		if strings.Contains(err.Error(), "cas conflict") {
			http.Error(w, "cas conflict", http.StatusPreconditionFailed)
			return
		}
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "key not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
