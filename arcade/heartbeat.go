package arcade

import (
	"encoding/json"
)

type HeartbeatMessage struct {
	Message

	Seq      int
	Metadata []byte
}

func NewHeartbeatMessage(seq int, metadata []byte) *HeartbeatMessage {
	return &HeartbeatMessage{
		Message:  Message{Type: "heartbeat"},
		Seq:      seq,
		Metadata: metadata,
	}
}

func (m HeartbeatMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
