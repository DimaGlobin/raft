package config

import (
	"os"
	"strings"
)

type Config struct {
	IsLeader bool
	Port     string
	Replicas []string
	Self     string
}

func Load() Config {
	return Config{
		IsLeader: os.Getenv("IS_LEADER") == "true",
		Port:     os.Getenv("PORT"),
		Replicas: strings.Split(os.Getenv("SLAVES"), ","), //TODO: переделать
	}
}
