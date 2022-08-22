package arcade

import (
	"arcade/arcade/message"
	"encoding/json"
)

type LeaveMessage struct {
	message.Message
	PlayerID string
	LobbyID  string
}

func NewLeaveMessage(playerID string, lobbyID string) *LeaveMessage {
	return &LeaveMessage{
		Message:  message.Message{Type: "leave"},
		PlayerID: playerID,
		LobbyID:  lobbyID,
	}
}

func (m LeaveMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
