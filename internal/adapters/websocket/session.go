package adapterwebsocket

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"time"

	"github.com/Tyulenb/asr-proxy/internal/model"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

var (
	wrongMessageType     = errors.New("Websocket message type is not expected type.")
	wrongTextMessageType = errors.New("Wrong text message type.")
	invalidAudioConfig   = errors.New("Audio config is not valid.")
)

type session struct {
	conn *websocket.Conn
}

func (s *session) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

func (s *session) ReadStart(ctx context.Context) (model.AudioConfig, error) {
	select {
	case <-ctx.Done():
		return model.AudioConfig{}, ctx.Err()
	default:
	}

	deadline := time.Now().Add(startWait)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := s.conn.SetReadDeadline(deadline); err != nil {
		return model.AudioConfig{}, err
	}

	readDone := make(chan struct{})
	defer close(readDone)
	go func() {
		select {
		case <-ctx.Done():
			s.Close()
		case <-readDone:
		}
	}()

	mt, msg, err := s.conn.ReadMessage()
	if err != nil {
		return model.AudioConfig{}, err
	}
	if err := s.conn.SetReadDeadline(time.Time{}); err != nil {
		return model.AudioConfig{}, err
	}
	if mt != websocket.TextMessage {
		return model.AudioConfig{}, wrongMessageType
	}
	txtMsg := new(model.ControlMessage)
	err = json.Unmarshal(msg, txtMsg)
	if err != nil {
		return model.AudioConfig{}, err
	}
	if txtMsg.Type != model.ControlMessageStart {
		return model.AudioConfig{}, wrongTextMessageType
	}
	audioCfg := model.AudioConfig{
		Format:     txtMsg.Format,
		SampleRate: txtMsg.SampleRate,
		Channels:   txtMsg.Channels,
	}
	if audioCfg.SampleRate <= 0 {
		return model.AudioConfig{}, invalidAudioConfig
	}
	if audioCfg.Channels <= 0 {
		return model.AudioConfig{}, invalidAudioConfig
	}
	return audioCfg, nil
}

func (s *session) Run(ctx context.Context, writer <-chan []byte, reader chan<- []byte) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errGroup, errCtx := errgroup.WithContext(runCtx)
	connectionDone := make(chan struct{})
	defer close(connectionDone)
	go func() {
		select {
		case <-errCtx.Done():
			s.Close()
		case <-connectionDone:
		}
	}()

	var normalCompletion atomic.Bool
	errGroup.Go(func() error {
		defer close(reader)
		defer cancel()
		err := s.receiveMessages(errCtx, reader)
		if err == nil {
			normalCompletion.Store(true)
		}
		return err
	})
	errGroup.Go(func() error {
		defer cancel()
		err := s.sendMessages(errCtx, writer)
		if err == nil {
			normalCompletion.Store(true)
		}
		return err
	})

	err := errGroup.Wait()
	if normalCompletion.Load() {
		return nil
	}
	return err
}

func (s *session) receiveMessages(ctx context.Context, reader chan<- []byte) error {
	s.conn.SetReadDeadline(time.Now().Add(pongWait))
	s.conn.SetPongHandler(func(string) error { s.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		mt, msg, err := s.conn.ReadMessage()
		if err != nil {
			return err
		}
		switch mt {
		case websocket.TextMessage:
			var textMsg model.ControlMessage
			err := json.Unmarshal(msg, &textMsg)
			if err != nil {
				return err
			}
			if textMsg.Type == model.ControlMessageStop {
				return nil
			} else {
				return wrongTextMessageType
			}
		case websocket.BinaryMessage:
			select {
			case <-ctx.Done():
				return ctx.Err()
			case reader <- msg:
			}
		default:
			return wrongMessageType
		}
	}
}

func (s *session) sendMessages(ctx context.Context, writer <-chan []byte) error {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-writer:
			if !ok {
				return nil
			}
			s.conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := s.conn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			s.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return err
			}
		}
	}
}
