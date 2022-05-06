package arcade

type ClientConnectEvent struct {
	ClientID string
}

func NewClientConnectEvent(clientID string) *ClientConnectEvent {
	return &ClientConnectEvent{
		ClientID: clientID,
	}
}
