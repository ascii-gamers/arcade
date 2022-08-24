package arcade

import (
	"arcade/arcade/net"
	"reflect"
)

func ProcessMessage(from *net.Client, msg interface{}, mgr *ViewManager) interface{} {
	// Get sender ID
	senderID := reflect.ValueOf(msg).FieldByName("Message").FieldByName("SenderID").String()
	sender, ok := arcade.Server.Network.GetClient(senderID)

	if !ok {
		panic("Unknown sender ID: " + senderID)
	}

	ret := mgr.ProcessMessage(sender, msg)

	mgr.RequestRender()

	return ret
}
