package config

import (
	"os"
	"strings"
)

type Config struct {
	Id    string
	Port  string
	Self  string
	Peers map[string]string
}

func Load() Config {
	peers := make(map[string]string)
	if env := os.Getenv("PEERS"); env != "" {
		for _, kv := range strings.Split(env, ",") {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) == 2 {
				peers[parts[0]] = parts[1]
			}
		}
	}

	return Config{
		Id:    os.Getenv("ID"),
		Port:  os.Getenv("PORT"),
		Self:  os.Getenv("SELF"),
		Peers: peers,
	}
}
