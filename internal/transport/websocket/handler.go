package websocketHandler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Tyulenb/asr-proxy/internal/model"
	"golang.org/x/sync/errgroup"
)

const (
	chanBufferSize = 100
)

type service interface {
	ProcessAudioChunk(context.Context, model.AudioConfig, chan<- []byte, <-chan []byte) error
}

type session interface {
	ReadStart(context.Context) (model.AudioConfig, error)
	Run(ctx context.Context, writer <-chan []byte, reader chan<- []byte) error
	Close()
}

type websocket interface {
	CreateSession(w http.ResponseWriter, r *http.Request) (session, error)
}

type WebsocketHandler struct {
	ws     websocket
	srv    service
	logger *slog.Logger
}

func NewWebsocketHandler(ws websocket, srv service) *WebsocketHandler {
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
	defer sess.Close()

	audioCfg, err := sess.ReadStart(r.Context())
	if err != nil {
		wh.logger.Error("Error during receiving audio config", "err", err)
		return
	}

	writer := make(chan []byte, chanBufferSize)
	reader := make(chan []byte, chanBufferSize)

	group, groupCtx := errgroup.WithContext(r.Context())
	group.Go(func() error {
		err := sess.Run(groupCtx, writer, reader)
		if err != nil {
			wh.logger.Error("Error during upgrading websocket", "err", err)
			return err
		}
		return nil
	})
	group.Go(func() error {
		err := wh.srv.ProcessAudioChunk(groupCtx, audioCfg, reader, writer)
		if err != nil {
			wh.logger.Error("Error during processing audio", "err", err)
			return err
		}
		return nil
	})
	_ = group.Wait()
}
