package arcade

import (
	"encoding/json"
)

type PongMessage struct {
	Message

	Distributor bool
}

func NewPongMessage(distributor bool) *PongMessage {
	return &PongMessage{
		Message:     Message{Type: "pong"},
		Distributor: distributor,
	}
}

func (m PongMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
