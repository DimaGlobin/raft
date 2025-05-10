package service

import (
	"encoding/json"
	"errors"

	"github.com/DimaGlobin/raft/internal/model/config"
	"github.com/DimaGlobin/raft/internal/repository"
	"github.com/DimaGlobin/raft/pkg/raft"
)

var _ Service = (*KvService)(nil)

type KvService struct {
	repo     repository.Repository
	isLeader bool
	replicas []string
	raft     raft.RaftNodeInterface
}

func NewKvService(cfg config.Config, repo repository.Repository, raft raft.RaftNodeInterface) Service {
	return &KvService{
		repo:     repo,
		isLeader: cfg.IsLeader,
		// replicas: cfg.Replicas,
		raft:     raft,
	}
}

func (s *KvService) applyOp(cmd raft.Command) error {
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	_, err = s.Raft().Apply(data)
	return err
}

func (s *KvService) Create(key string, value string) error {
	cmd := raft.Command{
		Op:    raft.Create,
		Id:    key,
		Value: value,
	}
	return s.applyOp(cmd)
}

func (s *KvService) Get(key string) (string, error) {
	return s.repo.Get(key)
}

func (s *KvService) Update(key, value string) error {
	cmd := raft.Command{
		Op:    raft.Update,
		Id:    key,
		Value: value,
	}
	return s.applyOp(cmd)
}

func (s *KvService) Patch(key, value string) error {
	_, err := s.repo.Get(key)
	if err != nil {
		return errors.New("key not found")
	}
	return s.Update(key, value)
}

func (s *KvService) Delete(key string) error {
	cmd := raft.Command{
		Op: raft.Delete,
		Id: key,
	}
	return s.applyOp(cmd)
}

func (s *KvService) GetReplicas() []string {
	return s.replicas
}

func (s *KvService) Raft() raft.RaftNodeInterface {
	return s.raft
}
