package arcade

import (
	"encoding/json"
)

type ClientsMessage struct {
	Message
	Clients map[string]int
}

func NewClientsMessage(clients map[string]int) *ClientsMessage {
	return &ClientsMessage{
		Message: Message{Type: "clients"},
		Clients: clients,
	}
}

func (m ClientsMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
