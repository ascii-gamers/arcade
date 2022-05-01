package arcade

import (
	"encoding/json"
)

type PongMessage struct {
	Message

	ID          string
	Clients     map[string]int
	Distributor bool
}

func NewPongMessage(id string, clients map[string]int, distributor bool) *PongMessage {
	return &PongMessage{
		Message:     Message{Type: "pong"},
		ID:          id,
		Clients:     clients,
		Distributor: distributor,
	}
}

func (m PongMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
