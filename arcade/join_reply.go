package arcade

import (
	"encoding/json"
)

const (
	OK           = "OK"
	ErrCapacity  = "ErrCapacity"
	ErrWrongCode = "ErrWrongCode"
)

type JoinErr string

type JoinReplyMessage struct {
	Message
	Error JoinErr
}

func NewJoinReplyMessage(err JoinErr) *JoinReplyMessage {
	return &JoinReplyMessage{
		Message: Message{Type: "join_reply"},
		Error:   err,
	}
}

func (m JoinReplyMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
