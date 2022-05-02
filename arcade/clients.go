package arcade

import (
	"encoding/json"
)

type ClientsMessage struct {
	Message
	Clients map[string]float64
}

func NewClientsMessage(clients map[string]float64) *ClientsMessage {
	return &ClientsMessage{
		Message: Message{Type: "clients"},
		Clients: clients,
	}
}

func (m ClientsMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
