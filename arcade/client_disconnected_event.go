package arcade

type ClientDisconnectedEvent struct {
	ClientID string
}

func NewClientDisconnectedEvent(clientID string) *ClientDisconnectedEvent {
	return &ClientDisconnectedEvent{
		ClientID: clientID,
	}
}
