package config

import (
	"flag"
	"github.com/caarlos0/env"
	"log"
)

type Configs struct {
	EnvLogs string `env:"LOG_LEVEL"`
}

func NewConfig() *Configs {
	var cfg Configs
	flag.StringVar(&cfg.EnvLogs, "l", "info", "Set logging level")
	flag.Parse()
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	return &cfg
}
