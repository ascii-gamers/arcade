package arcade

import (
	"arcade/arcade/message"
	"encoding/json"
)

type LobbyEndMessage struct {
	message.Message
	LobbyID string
}

func NewLobbyEndMessage(lobbyID string) *LobbyEndMessage {
	return &LobbyEndMessage{
		Message: message.Message{Type: "lobby_end"},
		LobbyID: lobbyID,
	}
}

func (m LobbyEndMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m LobbyEndMessage) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &m)
}
