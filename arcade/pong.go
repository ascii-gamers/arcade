package arcade

import (
	"encoding/json"
)

type PongMessage struct {
	Message

	ID          string
	Distributor bool
}

func NewPongMessage(id string, distributor bool) *PongMessage {
	return &PongMessage{
		Message:     Message{Type: "pong"},
		ID:          id,
		Distributor: distributor,
	}
}

func (m PongMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
