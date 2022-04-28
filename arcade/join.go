package arcade

import (
	"encoding/json"
)

type JoinMessage struct {
	Message
	Player Player
	Code   string
}

func NewJoinMessage(code string, player Player) *JoinMessage {
	return &JoinMessage{
		Message: Message{Type: "join"},
		Player:  player,
		Code:    code,
	}
}

func (m JoinMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
