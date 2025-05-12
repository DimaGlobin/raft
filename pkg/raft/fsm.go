package raft

import (
	"encoding/json"
	"fmt"

	"github.com/DimaGlobin/raft/internal/repository"
	"golang.org/x/exp/slog"
)

type CmdType string

const (
	Create CmdType = "create"
	Get    CmdType = "get"
	Update CmdType = "update"
	Delete CmdType = "delete"
	CAS    CmdType = "cas"
)

type Command struct {
	Op       CmdType `json:"op"`
	Id       string  `json:"id"`
	Value    string  `json:"value,omitempty"`
	Expected string  `json:"expected,omitempty"`
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
	store  repository.Repository
	logger *slog.Logger
}

func NewFSM(store repository.Repository, logger *slog.Logger) *FSM {
	return &FSM{
		store:  store,
		logger: logger,
	}
}

func (f *FSM) Apply(logData []byte) interface{} {
	var cmd Command
	if err := json.Unmarshal(logData, &cmd); err != nil {
		f.logger.Error("Failed to unmarshal command", "error", err)
		return err
	}

	f.logger.Debug("FSM Apply called", "op", cmd.Op, "key", cmd.Id, "val", cmd.Value)

	if err := cmd.Op.Valid(); err != nil {
		f.logger.Warn("Invalid command operation", "op", cmd.Op)
		return err
	}

	f.logger.Debug("Applying command", "op", cmd.Op, "id", cmd.Id, "value", cmd.Value)

	var err error
	switch cmd.Op {
	case Create:
		err = f.store.Create(cmd.Id, cmd.Value)
	case Update:
		err = f.store.Update(cmd.Id, cmd.Value)
	case Delete:
		err = f.store.Delete(cmd.Id)
	case CAS:
		curr, err := f.store.Get(cmd.Id)
		if err != nil {
			return fmt.Errorf("cas failed: key not found: %w", err)
		}
		if curr != cmd.Expected {
			return fmt.Errorf("cas conflict: expected %q, got %q", cmd.Expected, curr)
		}
		return f.store.Update(cmd.Id, cmd.Value)
	}

	if err != nil {
		f.logger.Error("Failed to apply command", "op", cmd.Op, "id", cmd.Id, "error", err)
		return err
	}

	f.logger.Info("Command applied", "op", cmd.Op, "id", cmd.Id)
	return nil
}
