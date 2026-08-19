package main

import (
	"github.com/Tyulenb/asr-proxy/internal/app"
	"github.com/Tyulenb/asr-proxy/internal/config"
)

func main() {
	cfg := config.NewConfig()
	api := app.NewApp(cfg)
	_ = api.Run()
}
