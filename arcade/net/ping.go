package net

import (
	"encoding/json"

	"arcade/arcade/message"
)

type PingMessage struct {
	message.Message
	Distributor bool
}

func NewPingMessage(distributor bool) *PingMessage {
	return &PingMessage{
		Message:     message.Message{Type: "ping"},
		Distributor: distributor,
	}
}

func (m PingMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m PingMessage) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &m)
}
