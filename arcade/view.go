package arcade

import (
	"arcade/arcade/net"
	"encoding"
)

type View interface {
	Init()
	GetHeartbeatMetadata() encoding.BinaryMarshaler
	ProcessEvent(ev interface{})
	ProcessMessage(from *net.Client, p interface{}) interface{}
	Render(s *Screen)
	Unload()
}
