package model

type ControlMessage struct {
	Type       ControlMessageType `json:"type"`
	SessionId  string             `json:"session_id"`
	SampleRate int                `json:"sample_rate,omitempty"`
	Channels   int                `json:"channels,omitempty"`
	Format     AudioFormat        `json:"format,omitempty"`
}

type ControlMessageType string

const (
	ControlMessageStart ControlMessageType = "start"
	ControlMessageStop  ControlMessageType = "stop"
)

type AudioFormat string

const (
	AudioFormatPCM16 AudioFormat = "pcm16"
	AudioFormatOPUS  AudioFormat = "opus"
	AudioFormatMP3   AudioFormat = "mp3"
	AudioFormatFLAC  AudioFormat = "flac"
)

type AudioConfig struct {
	SampleRate int         `json:"sample_rate,omitempty"`
	Channels   int         `json:"channels,omitempty"`
	Format     AudioFormat `json:"format,omitempty"`
}
