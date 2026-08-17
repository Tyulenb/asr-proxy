package infrawebsocket

import (
	"context"
	"errors"
	"net/http"

	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

const (
	ReadBufferSize  = 4096
	WriteBufferSize = 4096
)

var (
	wrongMessageType = errors.New("Websocket message type is not binary.")
)

type InfraWebsocket struct{}

func (iws *InfraWebsocket) CreateConnection(
	ctx context.Context,
	w http.ResponseWriter, r *http.Request,
	writer <-chan []byte, reader chan<- []byte,
) error {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  ReadBufferSize,
		WriteBufferSize: WriteBufferSize,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	errGroup, errCtx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		err := receiveMessages(errCtx, conn, reader)
		if err != nil {
			return err
		}
		return nil
	})
	errGroup.Go(func() error {
		err := sendMessages(errCtx, conn, writer)
		if err != nil {
			return err
		}
		return nil
	})
	return errGroup.Wait()
}

func receiveMessages(ctx context.Context, conn *websocket.Conn, reader chan<- []byte) error {
	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if mt != websocket.BinaryMessage {
			return wrongMessageType
		}
		reader <- msg
		select {
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func sendMessages(ctx context.Context, conn *websocket.Conn, writer <-chan []byte) error {
	for {
		select {
		case msg := <-writer:
			err := conn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
