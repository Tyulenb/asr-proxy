package app

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	grpcclient "github.com/Tyulenb/asr-proxy/internal/adapters/grpc-client"
	infrawebsocket "github.com/Tyulenb/asr-proxy/internal/adapters/websocket"
	"github.com/Tyulenb/asr-proxy/internal/config"
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
	conn, err := grpc.NewClient(a.cfg.ListenAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	gRPC := grpcclient.NewGrpcClient(conn, logger)
	ws := &infrawebsocket.Upgrader{}
	srvc := service.NewProxyService(logger, gRPC)
	handler := websocketHandler.NewWebsocketHandler(ws, srvc)

	router := http.NewServeMux()
	handler.RegisterRoutes(router)

	// TO DO: move to infrastructure layer
	// And add gracefull shutdown.
	server := http.Server{
		Addr:        ":9999",
		Handler:     router,
		ReadTimeout: time.Second * 10,
	}
	return server.ListenAndServe()
}
