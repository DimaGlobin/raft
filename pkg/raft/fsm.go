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
	Op    CmdType `json:"op"`
	Id    string  `json:"id"`
	Value string  `json:"value,omitempty"`
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

	var cmd Command
	if err := json.Unmarshal(logData, &cmd); err != nil {
		return err
	}

	if err := cmd.Op.Valid(); err != nil {
		return err
	}

	fmt.Println("APPLY:", cmd.Op, cmd.Id, cmd.Value)

	switch cmd.Op {
	case Create:
		return f.store.Create(cmd.Id, cmd.Value)
	case Update:
		return f.store.Update(cmd.Id, cmd.Value)
	case Delete:
		return f.store.Delete(cmd.Id)
	default:
		return fmt.Errorf("unsupported op: %s", cmd.Op)
	}
}
