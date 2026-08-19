package service

import (
	"context"
	"log/slog"

	"github.com/Tyulenb/asr-proxy/internal/model"
	"golang.org/x/sync/errgroup"
)

const (
	chanBufferSize = 100
)

type GrpcClient interface {
	Run(ctx context.Context, audioConfig model.AudioConfig, audio <-chan []byte, text chan<- []byte) error
	Close() error
}

type Session interface {
	ReadStart(context.Context) (model.AudioConfig, error)
	Run(ctx context.Context, writer <-chan []byte, reader chan<- []byte) error
	Close() error
}

type ProxyService struct {
	logger *slog.Logger
	grpc   GrpcClient
}

func NewProxyService(log *slog.Logger, grpc GrpcClient) *ProxyService {
	if log == nil {
		log = slog.Default()
	}
	return &ProxyService{
		logger: log,
		grpc:   grpc,
	}
}

func (ps *ProxyService) ProcessAudioChunks(ctx context.Context, sess Session) error {
	defer sess.Close()
	defer ps.grpc.Close()

	audioCfg, err := sess.ReadStart(ctx)
	if err != nil {
		ps.logger.Error("Error during receiving audio config", "err", err)
		return err
	}

	writer := make(chan []byte, chanBufferSize)
	reader := make(chan []byte, chanBufferSize)

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		err := sess.Run(groupCtx, writer, reader)
		if err != nil {
			ps.logger.Error("Error during upgrading websocket", "err", err)
			return err
		}
		return nil
	})
	group.Go(func() error {
		err := ps.grpc.Run(groupCtx, audioCfg, reader, writer)
		if err != nil {
			ps.logger.Error("Error during processing audio", "err", err)
			return err
		}
		return nil
	})
	return group.Wait()
}
