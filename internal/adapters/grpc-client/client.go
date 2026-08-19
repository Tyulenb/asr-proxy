package grpcclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"

	pb "github.com/Tyulenb/asr-proxy/internal/adapters/grpc-client/proto/v1"
	"github.com/Tyulenb/asr-proxy/internal/model"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type GrpcClient struct {
	conn   *grpc.ClientConn
	logger *slog.Logger
}

func NewGrpcClient(conn *grpc.ClientConn, logger *slog.Logger) *GrpcClient {
	if logger == nil {
		logger = slog.Default()
	}
	return &GrpcClient{
		conn:   conn,
		logger: logger,
	}
}

func (g *GrpcClient) Run(ctx context.Context, audioConfig model.AudioConfig, audio <-chan []byte, text chan<- []byte) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := pb.NewASRServiceClient(g.conn).RecognizeStream(runCtx)
	if err != nil {
		return fmt.Errorf("open stream %w", err)
	}
	defer stream.CloseSend()
	err = g.sendConfig(stream, audioConfig)
	if err != nil {
		return fmt.Errorf("send config %w", err)
	}

	errGroup, errCtx := errgroup.WithContext(runCtx)
	var normalCompletion atomic.Bool
	errGroup.Go(func() error {
		defer cancel()
		defer close(text)
		err := g.receiveResponses(errCtx, stream, text)
		if err == nil {
			normalCompletion.Store(true)
		}
		return err
	})
	errGroup.Go(func() error {
		defer cancel()
		err := g.sendAudio(errCtx, stream, audio)
		if err == nil {
			normalCompletion.Store(true)
		}
		return err
	})

	err = errGroup.Wait()
	if normalCompletion.Load() {
		return nil
	}
	return err
}

func (g *GrpcClient) Close() error {
	if g.conn != nil {
		return g.conn.Close()
	}
	return nil
}

func (g *GrpcClient) sendConfig(stream pb.ASRService_RecognizeStreamClient, audioCfg model.AudioConfig) error {
	var encoding pb.AudioConfig_AudioEncoding
	switch audioCfg.Format {
	case model.AudioFormatFLAC:
		encoding = pb.AudioConfig_AUDIO_ENCODING_FLAC
	case model.AudioFormatPCM16:
		encoding = pb.AudioConfig_AUDIO_ENCODING_PCM_16BIT
	case model.AudioFormatOPUS:
		encoding = pb.AudioConfig_AUDIO_ENCODING_OPUS
	case model.AudioFormatMP3:
		encoding = pb.AudioConfig_AUDIO_ENCODING_MP3
	default:
		encoding = pb.AudioConfig_AUDIO_ENCODING_UNSPECIFIED
	}
	err := stream.Send(&pb.RecognizeStreamRequest{
		Data: &pb.RecognizeStreamRequest_Config{
			Config: &pb.AudioConfig{
				Encoding:   encoding,
				SampleRate: int32(audioCfg.SampleRate),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("send audio configuration: %w", err)
	}
	return nil
}

func (g *GrpcClient) sendAudio(ctx context.Context, stream pb.ASRService_RecognizeStreamClient, audioChan <-chan []byte) error {
	for {
		g.logger.Debug("Sending chunks")
		select {
		case chunk, ok := <-audioChan:
			if !ok {
				return nil
			}
			err := stream.Send(&pb.RecognizeStreamRequest{
				Data: &pb.RecognizeStreamRequest_Chunk{Chunk: chunk},
			})
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (g *GrpcClient) receiveResponses(ctx context.Context, stream pb.ASRService_RecognizeStreamClient, text chan<- []byte) error {
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("receive recognition response: %w", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case text <- []byte(response.GetText()):
		}

		g.logger.Debug("Received text", "text", response.GetText())
	}
}
