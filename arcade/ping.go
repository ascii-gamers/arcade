package arcade

import (
	"encoding/json"
)

type PingMessage struct {
	Message
}

func NewPingMessage() *PingMessage {
	return &PingMessage{
		Message: Message{Type: "ping"},
	}
}

func (m PingMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
