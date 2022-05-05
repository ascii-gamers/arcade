package arcade

import (
	"encoding/json"
)

type HeartbeatReplyMessage struct {
	Message

	Seq int
}

func NewHeartbeatReplyMessage(seq int) *HeartbeatReplyMessage {
	return &HeartbeatReplyMessage{
		Message: Message{Type: "heartbeat_reply"},
		Seq:     seq,
	}
}

func (m HeartbeatReplyMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
