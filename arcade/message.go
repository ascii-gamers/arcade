package arcade

import (
	"encoding/json"
	"fmt"
)

type Message struct {
	Type string `json:"type"`
}

func processMessage(from *Client, data []byte) interface{} {
	p, err := parseMessage(data)

	if err != nil {
		return err
	}

	ret := mgr.view.ProcessPacket(p)

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
	case "error":
		p := ErrorMessage{}
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
	default:
		return nil, fmt.Errorf("Unknown message type '%s'", res.Type)
	}
}
