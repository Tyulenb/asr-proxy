package app

import "github.com/Tyulenb/asr-proxy/internal/config"

type App struct {
	cfg *config.Config
}

func NewApp(cfg *config.Config) *App {
	return &App{
		cfg: cfg,
	}
}
