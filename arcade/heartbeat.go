package arcade

import (
	"encoding/json"
)

type HeartbeatMessage struct {
	Message

	Seq int
}

func NewHeartbeatMessage(seq int) *HeartbeatMessage {
	return &HeartbeatMessage{
		Message: Message{Type: "heartbeat"},
		Seq:     seq,
	}
}

func (m HeartbeatMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
