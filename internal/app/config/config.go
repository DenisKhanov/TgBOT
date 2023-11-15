package config

import (
	"flag"
	"github.com/caarlos0/env"
	"log"
)

type Configs struct {
	EnvLogs        string `env:"LOG_LEVEL"`
	EnvStoragePath string `env:"FILE_STORAGE_PATH"`
	EnvToken       string `env:"TOKEN"`
}

func NewConfig() *Configs {
	var cfg Configs
	flag.StringVar(&cfg.EnvLogs, "l", "info", "Set logging level")
	flag.StringVar(&cfg.EnvStoragePath, "f", "/tmp/tgBot-db.json", "Path for saving data file")
	flag.StringVar(&cfg.EnvToken, "t", "", "BOT TOKEN")
	flag.Parse()
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	return &cfg
}
