package arcade

import (
	"encoding/json"
)

type LobbyInfoMessage struct {
	Message
	GameInfo *PendingGame

	IP string
}

func NewLobbyInfoMessage(game *PendingGame, ip string) *LobbyInfoMessage {
	return &LobbyInfoMessage{
		Message:  Message{Type: "lobby_info"},
		GameInfo: game,
		IP:       ip,
	}
}

func (m LobbyInfoMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
