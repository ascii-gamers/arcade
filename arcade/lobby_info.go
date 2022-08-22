package arcade

import (
	"arcade/arcade/message"
	"encoding/json"
)

type LobbyInfoMessage struct {
	message.Message
	Lobby *Lobby
}

func NewLobbyInfoMessage(lobby *Lobby) *LobbyInfoMessage {
	return &LobbyInfoMessage{
		Message: message.Message{Type: "lobby_info"},
		Lobby:   lobby,
	}
}

func (m LobbyInfoMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m LobbyInfoMessage) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &m)
}
