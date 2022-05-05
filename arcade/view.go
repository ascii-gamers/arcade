package arcade

import "encoding"

type View interface {
	Init()
	GetHeartbeatMetadata() encoding.BinaryMarshaler
	ProcessEvent(ev interface{})
	ProcessMessage(from *Client, p interface{}) interface{}
	Render(s *Screen)
	Unload()
}
