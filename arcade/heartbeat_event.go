package arcade

type HeartbeatEvent struct {
	Metadata []byte
}

func NewHeartbeatEvent(metadata []byte) *HeartbeatEvent {
	return &HeartbeatEvent{
		Metadata: metadata,
	}
}
