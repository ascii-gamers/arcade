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
	sender, ok := arcade.Server.Network.GetClient(senderID)

	if !ok {
		panic("Unknown sender ID: " + senderID)
	}

	ret := arcade.ViewManager.ProcessMessage(sender, p)

	arcade.ViewManager.RequestRender()

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
	case "client_update":
		p := ClientUpdateMessage[TronClientState]{}
		json.Unmarshal(data, &p)
		return p, nil
	case "disconnect":
		p := DisconnectMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	case "error":
		p := ErrorMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	case "game_update":
		p := GameUpdateMessage[TronGameState, TronClientState]{}
		json.Unmarshal(data, &p)
		return p, nil
	case "heartbeat":
		p := HeartbeatMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	case "heartbeat_reply":
		p := HeartbeatReplyMessage{}
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
	case "lobby_end":
		p := LobbyEndMessage{}
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
	case "start_game":
		p := StartGameMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	case "ack_game_update":
		p := AckGameUpdateMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	case "end_game":
		p := EndGameMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	default:
		return nil, fmt.Errorf("Unknown message type '%s'", res.Type)
	}
}
