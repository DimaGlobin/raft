package service

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/DimaGlobin/raft/internal/repository"
	"github.com/DimaGlobin/raft/pkg/raft"
	"golang.org/x/exp/slog"
)

var _ Service = (*KvService)(nil)

type KvService struct {
	repo     repository.Repository
	isLeader bool
	replicas []string
	raft     raft.RaftNodeInterface
	logger   *slog.Logger
}

func NewKvService(repo repository.Repository, raft raft.RaftNodeInterface, log *slog.Logger) Service {
	return &KvService{
		repo:   repo,
		raft:   raft,
		logger: log,
	}
}

func (s *KvService) applyOp(cmd raft.Command) error {
	s.logger.Debug("applyOp called", "cmd", cmd)

	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	s.logger.Debug("calling ApplyWithResult")
	result, err := s.raft.ApplyWithResult(data, 2*time.Second)
	if err != nil {
		s.logger.Error("ApplyWithResult failed", "error", err)
		return err
	}

	s.logger.Debug("apply result received", "result", result)

	if err, ok := result.(error); ok {
		return err
	}

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

func (s *KvService) CAS(key, expected, newVal string) error {
	cmd := raft.Command{
		Op:       raft.CAS,
		Id:       key,
		Value:    newVal,
		Expected: expected,
	}
	return s.applyOp(cmd)
}
