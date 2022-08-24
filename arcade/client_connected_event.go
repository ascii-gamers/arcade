package arcade

type ClientConnectedEvent struct {
	ClientID string
}

func NewClientConnectedEvent(clientID string) *ClientConnectedEvent {
	return &ClientConnectedEvent{
		ClientID: clientID,
	}
}
