package config

import "github.com/caarlos0/env/v11"

type Config struct {
	ListenAddress string `env:"LISTEN_ADDRESS" envDefault:"6666"`
}

func NewConfig() *Config {
	var cfg Config
	env.Parse(&cfg)
	return &cfg
}
