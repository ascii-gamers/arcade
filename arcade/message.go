package arcade

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type Message struct {
	SenderID    string
	RecipientID string
	MessageID   string
	Type        string
}

func processMessage(from *Client, p interface{}) interface{} {
	// Get sender ID
	senderID := reflect.ValueOf(p).FieldByName("Message").FieldByName("SenderID").String()
	sender, ok := server.Network.GetClient(senderID)

	if !ok {
		panic("Unknown sender ID: " + senderID)
	}

	ret := mgr.view.ProcessMessage(sender, p)

	mgr.RequestRender()

	return ret
}

func parseMessage(data []byte) (interface{}, error) {
	res := struct {
		Type string
	}{}

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	switch res.Type {
	case "clients":
		p := ClientsMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	case "disconnect":
		p := DisconnectMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	case "distance_update":
		p := DistanceUpdateMessage{}
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
		p := JoinReplyMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	case "leave":
		p := LeaveMessage{}
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
	case "routing":
		p := RoutingMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	default:
		return nil, fmt.Errorf("Unknown message type '%s'", res.Type)
	}
}
