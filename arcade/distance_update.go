package arcade

import "encoding/json"

type DistanceUpdateMessage struct {
	Message

	Adjacency map[string]float64
}

func NewDistanceUpdateMessage() *DistanceUpdateMessage {
	return &DistanceUpdateMessage{
		Message: Message{Type: "distance_update"},
	}
}

func (m DistanceUpdateMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
