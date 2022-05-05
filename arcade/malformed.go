package arcade

import (
	"encoding/json"
)

// Used as a placeholder when there's an invalid packet
type MalformedMessage struct {
	Message

	Text string
}

func NewMalformedMessage(text string) *MalformedMessage {
	return &MalformedMessage{
		Message: Message{Type: "malformed"},
		Text:    text,
	}
}

func (m MalformedMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
