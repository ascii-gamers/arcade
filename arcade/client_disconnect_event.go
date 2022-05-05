package arcade

type ClientDisconnectEvent struct {
	ClientID string
}

func NewClientDisconnectEvent(clientID string) *ClientDisconnectEvent {
	return &ClientDisconnectEvent{
		ClientID: clientID,
	}
}
