package arcade

import (
	"encoding/json"
)

type GetClientsMessage struct {
	Message
}

func NewGetClientsMessage() *GetClientsMessage {
	return &GetClientsMessage{
		Message: Message{Type: "get_clients"},
	}
}

func (m GetClientsMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
