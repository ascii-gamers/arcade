package arcade

import (
	"encoding/json"
)

type LeaveMessage struct {
	Message
	PlayerID string
}

func NewLeaveMessage(playerId string) *LeaveMessage {
	return &LeaveMessage{
		Message:  Message{Type: "leave"},
		PlayerID: playerId,
	}
}

func (m LeaveMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
