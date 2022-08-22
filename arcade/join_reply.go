package arcade

import (
	"arcade/arcade/message"
	"encoding/json"
)

const (
	OK           = "OK"
	ErrCapacity  = "ErrCapacity"
	ErrWrongCode = "ErrWrongCode"
)

type JoinErr string

type JoinReplyMessage struct {
	message.Message
	Lobby *Lobby
	Error JoinErr
}

func NewJoinReplyMessage(lobby *Lobby, err JoinErr) *JoinReplyMessage {
	return &JoinReplyMessage{
		Message: message.Message{Type: "join_reply"},
		Lobby:   lobby,
		Error:   err,
	}
}

func (m JoinReplyMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
