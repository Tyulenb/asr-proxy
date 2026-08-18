package model

type ControlMessage struct {
	Type       ControlMessageType `json:"type"`
	SessionId  string             `json:"session_id"`
	SampleRate int                `json:"sample_rate,omitempty"`
	Channels   int                `json:"channels,omitempty"`
	Format     string             `json:"format,omitempty"`
}

type ControlMessageType string

const (
	ControlMessageStart ControlMessageType = "start"
	ControlMessageStop  ControlMessageType = "stop"
)

type AudioConfig struct {
	SampleRate int    `json:"sample_rate,omitempty"`
	Channels   int    `json:"channels,omitempty"`
	Format     string `json:"format,omitempty"`
}
