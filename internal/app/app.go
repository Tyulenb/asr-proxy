package app

import (
	"log/slog"
	"os"

	grpcclient "github.com/Tyulenb/asr-proxy/internal/adapters/grpc-client"
	adapterwebsocket "github.com/Tyulenb/asr-proxy/internal/adapters/websocket"
	"github.com/Tyulenb/asr-proxy/internal/config"
	infrahttp "github.com/Tyulenb/asr-proxy/internal/infra/http"
	"github.com/Tyulenb/asr-proxy/internal/service"
	websocketHandler "github.com/Tyulenb/asr-proxy/internal/transport/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type App struct {
	cfg *config.Config
}

func NewApp(cfg *config.Config) *App {
	return &App{
		cfg: cfg,
	}
}

func (a *App) Run() error {
	conn, err := grpc.NewClient(a.cfg.ASRAdress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	gRPC := grpcclient.NewGrpcClient(conn, logger)
	ws := &adapterwebsocket.Upgrader{}
	srvc := service.NewProxyService(logger, gRPC)
	handler := websocketHandler.NewWebsocketHandler(ws, srvc)

	logger.Info("Server is running", "address", a.cfg.ListenAddress)

	err = infrahttp.ServerRun(a.cfg, handler)
	if err != nil {
		logger.Error("Error during server work", "err", err)
		return err
	}
	return nil
}
