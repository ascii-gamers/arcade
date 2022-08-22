package net

import (
	"encoding/json"

	"arcade/arcade/message"
)

type PingMessage struct {
	message.Message
}

func NewPingMessage() *PingMessage {
	return &PingMessage{
		Message: message.Message{Type: "ping"},
	}
}

func (m PingMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m PingMessage) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &m)
}
