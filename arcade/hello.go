package arcade

import (
	"encoding/json"
	"fmt"
)

type HelloMessage struct {
	Message
}

func HelloMessageHandler(client *Client, p HelloMessage) interface{} {
	fmt.Println("received hello!")
	return nil
}

func NewHelloMessage() *HelloMessage {
	return &HelloMessage{
		Message: Message{"hello"},
	}
}

func (m HelloMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}
