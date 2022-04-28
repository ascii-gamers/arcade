package arcade

import (
	"encoding/json"
	"fmt"
)

type Message struct {
	SenderID    string `json:"sender_id"`
	RecipientID string `json:"recipient_id"`
	ID          string `json:"id"`
	Type        string `json:"type"`
}

func processMessage(from *Client, p interface{}) interface{} {
	ret := mgr.view.ProcessPacket(from, p)

	mgr.view.Render(mgr.screen)
	mgr.screen.Show()

	return ret
}

func parseMessage(data []byte) (interface{}, error) {
	res := struct {
		Type string `json:"type"`
	}{}

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	switch res.Type {
	case "clients":
		p := ClientsMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	case "error":
		p := ErrorMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	case "get_clients":
		p := GetClientsMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	case "hello":
		p := HelloMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	case "lobby_info":
		p := LobbyInfoMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	case "join":
		p := JoinMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	case "join_reply":
		p := JoinMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	case "ping":
		p := PingMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	case "pong":
		p := PongMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	default:
		return nil, fmt.Errorf("Unknown message type '%s'", res.Type)
	}
}
