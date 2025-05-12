# Raft Consensus Implementation in Go

This repository contains a minimal, educational implementation of the [Raft consensus algorithm](https://raft.github.io/) in Go.  
It supports leader election, log replication, and state machine application.

## 📦 Features

- ✅ Leader election via Raft protocol  
- ✅ Log replication across nodes  
- ✅ Commit index and quorum logic  
- ✅ In-memory key-value state machine  
- ✅ HTTP-based peer communication  
- ✅ Docker-based multi-node testing setup  
- ✅ Transparent logging with `slog`

## 📚 Project Structure

```
.
├── cmd/                  # CLI entrypoint
├── raft/                 # Core Raft logic
│   ├── node.go           # Main Raft node logic
│   ├── fsm.go            # FSM for command application
│   └── ...
├── internal/             # Middleware, service, repository and handlers
├── Dockerfile
├── docker-compose.yml    # Multi-node Raft cluster setup
├── README.md
└── Makefile

```

## 🚀 Getting Started

### Prerequisites

- Go 1.23+
- Docker + Docker Compose

### Run Cluster (3 Nodes)

```bash
make run
```

This starts 3 Raft nodes:
- `node1` at `localhost:8081`
- `node2` at `localhost:8082`
- `node3` at `localhost:8083`

### Interact with the Cluster

**Write a value (only to leader):**

```bash
curl -X POST localhost:8081/kv -d '{"key": "foo", "value": "bar"}'
```

**Read a value (from any node):**

```bash
curl localhost:8082/kv?key=foo
```