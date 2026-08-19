package websocketHandler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Tyulenb/asr-proxy/internal/service"
)

const (
	chanBufferSize = 100
)

type HandlerService interface {
	ProcessAudioChunks(ctx context.Context, sess service.Session) error
}

type websocket interface {
	CreateSession(w http.ResponseWriter, r *http.Request) (service.Session, error)
}

type WebsocketHandler struct {
	ws     websocket
	srv    HandlerService
	logger *slog.Logger
}

func NewWebsocketHandler(ws websocket, srv HandlerService) *WebsocketHandler {
	return &WebsocketHandler{
		ws:  ws,
		srv: srv,
	}
}

func (wh *WebsocketHandler) RegisterRoutes(router *http.ServeMux) {
	router.HandleFunc("/voice", wh.audioChunks)
}

func (wh *WebsocketHandler) audioChunks(w http.ResponseWriter, r *http.Request) {
	sess, err := wh.ws.CreateSession(w, r)
	if err != nil {
		return
	}
	err = wh.srv.ProcessAudioChunks(r.Context(), sess)
	if err != nil {
		wh.logger.Error("Error occurs during audio processing", "err", err)
		return
	}
	wh.logger.Debug("Audio processing completed successfully!")
}
