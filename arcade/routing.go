package arcade

import "encoding/json"

type RoutingMessage struct {
	Message

	Distances map[string]*ClientRoutingInfo
}

func NewRoutingMessage(distances map[string]*ClientRoutingInfo) *RoutingMessage {
	return &RoutingMessage{
		Message:   Message{Type: "routing"},
		Distances: distances,
	}
}

func (m RoutingMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
