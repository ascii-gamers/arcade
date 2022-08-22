package arcade

import (
	"arcade/arcade/message"
	"encoding/json"
)

type ErrorMessage struct {
	message.Message

	Text string
}

func NewErrorMessage(msg string) *ErrorMessage {
	return &ErrorMessage{
		Message: message.Message{Type: "error"},
		Text:    msg,
	}
}

func (m ErrorMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
