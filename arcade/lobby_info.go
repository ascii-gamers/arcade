package arcade

import (
	"encoding/json"
)

type LobbyInfoMessage struct {
	Message

	IP string
}

func NewLobbyInfoMessage(ip string) *LobbyInfoMessage {
	return &LobbyInfoMessage{
		Message: Message{Type: "lobby_info"},
		IP:      ip,
	}
}

func (m LobbyInfoMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
