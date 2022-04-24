package arcade

import (
	"encoding/json"
)

type PongMessage struct {
	Message
}

func NewPongMessage() *PongMessage {
	return &PongMessage{
		Message: Message{Type: "pong"},
	}
}

func (m PongMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
