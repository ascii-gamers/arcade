package net

import (
	"encoding/json"

	"arcade/arcade/message"
)

type PongMessage struct {
	message.Message

	Distributor bool
}

func NewPongMessage(distributor bool) *PongMessage {
	return &PongMessage{
		Message:     message.Message{Type: "pong"},
		Distributor: distributor,
	}
}

func (m PongMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m PongMessage) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &m)
}
