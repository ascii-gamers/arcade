package arcade

import (
	"encoding/json"
)

type JoinMessage struct {
	Message
	PlayerID string
	Code   string
}

func NewJoinMessage(code string, playerID string) *JoinMessage {
	return &JoinMessage{
		Message: Message{Type: "join"},
		PlayerID:  playerID,
		Code:    code,
	}
}

func (m JoinMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
