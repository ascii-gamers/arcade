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
	Lobby *Lobby
	Error JoinErr
}

func NewJoinReplyMessage(lobby *Lobby, err JoinErr) *JoinReplyMessage {
	return &JoinReplyMessage{
		Message: Message{Type: "join_reply"},
		Lobby:   lobby,
		Error:   err,
	}
}

func (m JoinReplyMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
