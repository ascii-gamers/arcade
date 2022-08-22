package arcade

import (
	"arcade/arcade/message"
	"encoding/json"
)

type HelloMessage struct {
	message.Message
}

func NewHelloMessage() *HelloMessage {
	return &HelloMessage{
		Message: message.Message{Type: "hello"},
	}
}

func (m HelloMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m HelloMessage) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &m)
}
