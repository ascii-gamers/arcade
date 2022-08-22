package message

import "encoding/json"

type Message struct {
	SenderID    string
	RecipientID string
	MessageID   string
	Type        string
}

func (m Message) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &m)
}
