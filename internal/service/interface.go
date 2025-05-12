package service

import "github.com/DimaGlobin/raft/pkg/raft"

type Service interface {
	Delete(key string) error
	Get(key string) (string, error)
	Update(key, val string) error
	Patch(key, value string) error
	Create(key, value string) error
	GetReplicas() []string
	Raft() raft.RaftNodeInterface
	CAS(key, expected, newVal string) error
}
