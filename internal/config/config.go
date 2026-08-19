package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ListenAddress string        `env:"LISTEN_ADDRESS" envDefault:":6666"`
	ASRAdress     string        `env:"ASR_ADRESS" envDefault:":50051"`
	ShutdownGrace time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"5s"`
	ReadTimeout   time.Duration `env:"READ_TIMEOUT" envDefault:"2s"`
	WriteTimeout  time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`
}

func NewConfig() *Config {
	var cfg Config
	env.Parse(&cfg)
	return &cfg
}
