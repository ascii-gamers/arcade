package arcade

import (
	"encoding/json"
)

type PingMessage struct {
	Message

	ID string `json:"id"`
}

func NewPingMessage(id string) *PingMessage {
	return &PingMessage{
		Message: Message{Type: "ping"},
		ID:      id,
	}
}

func (m PingMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
