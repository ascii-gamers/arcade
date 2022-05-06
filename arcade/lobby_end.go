package arcade

import (
	"encoding/json"
)

type LobbyEndMessage struct {
	Message
	LobbyID string
}

func NewLobbyEndMessage(lobbyID string) *LobbyEndMessage {
	return &LobbyEndMessage{
		Message: Message{Type: "lobby_end"},
		LobbyID: lobbyID,
	}
}

func (m LobbyEndMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
