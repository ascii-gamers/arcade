package arcade

import (
	"encoding/json"
)

type JoinMessage struct {
	Message
	PlayerID string
	Code     string
	LobbyID  string
}

func NewJoinMessage(code string, playerID string, lobbyID string) *JoinMessage {
	return &JoinMessage{
		Message:  Message{Type: "join"},
		PlayerID: playerID,
		Code:     code,
		LobbyID:  lobbyID,
	}
}

func (m JoinMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
