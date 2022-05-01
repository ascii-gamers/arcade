package arcade

import (
	"encoding/json"
)

type DisconnectMessage struct {
	Message
}

func NewDisconnectMessage() *DisconnectMessage {
	return &DisconnectMessage{
		Message: Message{Type: "disconnect"},
	}
}

func (m DisconnectMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
