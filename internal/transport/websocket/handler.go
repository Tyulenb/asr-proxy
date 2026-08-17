package websocket

import (
	"context"
	"log/slog"
	"net/http"

	"golang.org/x/sync/errgroup"
)

const (
	chanBufferSize = 100
)

type service interface {
	ProcessAudioChunk(context.Context, chan<- []byte, <-chan []byte) error
}

type websocket interface {
	// Connection will read data from client into reader chan.
	// And will write data to client from writer chan.
	CreateConnection(
		ctx context.Context,
		w http.ResponseWriter, r *http.Request,
		writer <-chan []byte, reader chan<- []byte,
	) error
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
	writer := make(chan []byte, chanBufferSize)
	reader := make(chan []byte, chanBufferSize)

	group, groupCtx := errgroup.WithContext(r.Context())
	group.Go(func() error {
		err := wh.ws.CreateConnection(groupCtx, w, r, writer, reader)
		if err != nil {
			wh.logger.Error("Error during upgrading websocket", "err", err)
			return err
		}
		return nil
	})
	group.Go(func() error {
		err := wh.srv.ProcessAudioChunk(groupCtx, reader, writer)
		if err != nil {
			wh.logger.Error("Error during processing audio", "err", err)
			return err
		}
		return nil
	})
	err := group.Wait()
	if err != nil {
		writeErrorResponse(w)
	}
}

func writeErrorResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Something went wrong"))
}
