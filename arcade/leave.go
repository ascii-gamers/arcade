package arcade

import (
	"encoding/json"
)

type LeaveMessage struct {
	Message
	Player Player
}

func NewLeaveMessage(player Player) *LeaveMessage {
	return &LeaveMessage{
		Message: Message{Type: "leave"},
		Player:  player,
	}
}

func (m LeaveMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
