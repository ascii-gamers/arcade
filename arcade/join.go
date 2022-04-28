package arcade

import (
	"encoding/json"
)

const (
	OK             = "OK"
	ErrCapacity = "ErrCapacity"
	ErrWrongCode = "ErrWrongCode"
)

type JoinErr string

type JoinMessage struct {
	Message
	Code string
}

type JoinReplyMessage struct {
	Message
	Error JoinErr
}

func NewJoinMessage(code string) *JoinMessage {
	return &JoinMessage{
		Message: Message{Type: "join"},
		Code: code,
	}
}

func (m JoinMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func NewJoinReplyMessage(err JoinErr) *JoinReplyMessage {
	return &JoinReplyMessage{
		Message: Message{Type: "join"},
		Error: err,
	}
}

func (m JoinReplyMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

