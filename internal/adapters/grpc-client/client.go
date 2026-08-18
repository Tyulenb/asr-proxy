package grpcclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	pb "github.com/Tyulenb/asr-proxy/internal/adapters/grpc-client/proto/v1"
	"github.com/Tyulenb/asr-proxy/internal/model"
	"google.golang.org/grpc"
)

type GrpcClient struct {
	conn   *grpc.ClientConn
	logger *slog.Logger
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
				err := stream.CloseSend()
				return err
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

		g.logger.Debug("Received tezt", "text", response.GetText())
	}
}
