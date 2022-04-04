package arcade

import (
	"encoding/json"
	"fmt"
)

type ErrorMessage struct {
	Message

	Text string `json:"text"`
}

func ErrorMessageHandler(client *Client, p ErrorMessage) interface{} {
	fmt.Println("received error:", p.Text)
	return nil
}

func NewErrorMessage(msg string) *ErrorMessage {
	return &ErrorMessage{
		Message: Message{"error"},
		Text:    msg,
	}
}

func (m ErrorMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
