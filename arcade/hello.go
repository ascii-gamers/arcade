package arcade

import (
	"encoding/json"
)

type HelloMessage struct {
	Message
}

func NewHelloMessage() *HelloMessage {
	return &HelloMessage{
		Message: Message{"hello"},
	}
}

func (m HelloMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
