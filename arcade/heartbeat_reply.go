package arcade

import (
	"arcade/arcade/message"
	"encoding/json"
)

type HeartbeatReplyMessage struct {
	message.Message

	Seq int
}

func NewHeartbeatReplyMessage(seq int) *HeartbeatReplyMessage {
	return &HeartbeatReplyMessage{
		Message: message.Message{Type: "heartbeat_reply"},
		Seq:     seq,
	}
}

func (m HeartbeatReplyMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
