package arcade

import (
	"arcade/arcade/message"
	"encoding/json"
)

type DisconnectMessage struct {
	message.Message
}

func NewDisconnectMessage() *DisconnectMessage {
	return &DisconnectMessage{
		Message: message.Message{Type: "disconnect"},
	}
}

func (m DisconnectMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m DisconnectMessage) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &m)
}
