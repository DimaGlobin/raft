package service

import (
	"errors"
	"time"

	"github.com/DimaGlobin/raft/internal/model/config"
	"github.com/DimaGlobin/raft/internal/repository"
	"github.com/DimaGlobin/raft/pkg/raft"
)

var _ Service = (*KvService)(nil)

type KvService struct {
	repo     repository.Repository
	isLeader bool
	replicas []string
	raft     *raft.Node
}

func New(cfg config.Config, repo repository.Repository) Service {
	return &KvService{
		repo:     repo,
		isLeader: cfg.IsLeader,
		replicas: cfg.Replicas,
	}
}

func (s *KvService) applyOp(cmd raft.Command) error {
	return s.Raft().Apply(b, 5*time.Second)
}

func (s *KvService) Create(key, value string) error {
	return s.repo.Create(key, value)
}

func (s *KvService) Get(key string) (string, error) {
	return s.repo.Get(key)
}

func (s *KvService) Update(key, value string) error {
	return s.repo.Update(key, value)
}

func (s *KvService) Patch(key, value string) error {
	_, err := s.repo.Get(key)
	if err != nil {
		return errors.New("key not found")
	}
	return s.repo.Update(key, value)
}

func (s *KvService) Delete(key string) error {
	return s.repo.Delete(key)
}

func (s *KvService) GetReplicas() []string {
	return s.replicas
}

func (s *KvService) Raft() *raft.Node {
	return s.raft
}
