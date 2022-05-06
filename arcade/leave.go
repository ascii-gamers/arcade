package arcade

import (
	"encoding/json"
)

type LeaveMessage struct {
	Message
	PlayerID string
	LobbyID  string
}

func NewLeaveMessage(playerID string, lobbyID string) *LeaveMessage {
	return &LeaveMessage{
		Message:  Message{Type: "leave"},
		PlayerID: playerID,
		LobbyID:  lobbyID,
	}
}

func (m LeaveMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
