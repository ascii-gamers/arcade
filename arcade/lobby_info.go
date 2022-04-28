package arcade

import (
	"encoding/json"
)

type LobbyInfoMessage struct {
	Message
	Lobby *Lobby
}

func NewLobbyInfoMessage(lobby *Lobby) *LobbyInfoMessage {
	return &LobbyInfoMessage{
		Message: Message{Type: "lobby_info"},
		Lobby:   lobby,
	}
}

func (m LobbyInfoMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
