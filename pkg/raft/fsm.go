package raft

import (
	"encoding/json"
	"fmt"

	"github.com/DimaGlobin/raft/internal/repository"
)

type CmdType string

const (
	Create CmdType = "create"
	Get    CmdType = "get"
	Update CmdType = "update"
	Delete CmdType = "delete"
)

type Command struct {
	Op    string `json:"op"`
	Id    string `json:"id"`
	Value []byte `json:"value,omitempty"`
}

func (c CmdType) Valid() error {
	switch c {
	case Create, Get, Update, Delete:
		return nil
	default:
		return fmt.Errorf("unknown cmd type: %s", c)
	}
}

type FSM struct {
	store repository.Repository
}

func NewFSM(store repository.Repository) *FSM {
	return &FSM{store: store}
}

func (f *FSM) Apply(logData []byte) interface{} {
	var cmd struct {
		Op    CmdType `json:"op"`
		ID    string  `json:"id"`
		Value string  `json:"value,omitempty"`
	}
	if err := json.Unmarshal(logData, &cmd); err != nil {
		return err
	}

	if err := cmd.Op.Valid(); err != nil {
		return err
	}

	switch cmd.Op {
	case "create":
		return f.store.Create(cmd.ID, cmd.Value)
	case "update":
		return f.store.Update(cmd.ID, cmd.Value)
	case "delete":
		return f.store.Delete(cmd.ID)
	default:
		return nil
	}
}
