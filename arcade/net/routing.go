package net

import (
	"encoding/json"

	"arcade/arcade/message"
)

type RoutingMessage struct {
	message.Message

	Distances map[string]ClientRoutingInfo
}

func NewRoutingMessage(distances map[string]ClientRoutingInfo) *RoutingMessage {
	return &RoutingMessage{
		Message:   message.Message{Type: "routing"},
		Distances: distances,
	}
}

func (m RoutingMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m RoutingMessage) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &m)
}
