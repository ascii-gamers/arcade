package arcade

import (
	"encoding/json"
)

type PongMessage struct {
	Message

	ID          string             `json:"id"`
	Clients     map[string]float64 `json:"clients"`
	Distributor bool               `json:"distributor"`
}

func NewPongMessage(id string, clients map[string]float64, distributor bool) *PongMessage {
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
