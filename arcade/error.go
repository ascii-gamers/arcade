package arcade

import (
	"encoding/json"
)

type ErrorMessage struct {
	Message

	Text string `json:"text"`
}

func NewErrorMessage(msg string) *ErrorMessage {
	return &ErrorMessage{
		Message: Message{Type: "error"},
		Text:    msg,
	}
}

func (m ErrorMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
