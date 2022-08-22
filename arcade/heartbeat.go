package arcade

import (
	"arcade/arcade/message"
	"encoding/json"
)

type HeartbeatMessage struct {
	message.Message

	Seq      int
	Metadata []byte
}

func NewHeartbeatMessage(seq int, metadata []byte) *HeartbeatMessage {
	return &HeartbeatMessage{
		Message:  message.Message{Type: "heartbeat"},
		Seq:      seq,
		Metadata: metadata,
	}
}

func (m HeartbeatMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
