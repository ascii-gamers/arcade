package message

import (
	"encoding/json"
	"log"
	"reflect"
)

type Listener struct {
	// One listener is the distributor listener, and handles forwarding
	// messages to the correct client. All others should have this set to false
	Distributor bool

	ServerID string

	// The function to call when a message is received
	Handle func(c, data interface{}) interface{}
}

var listeners = make([]Listener, 0)

func AddListener(listener Listener) {
	listeners = append(listeners, listener)
}

func Notify(c interface{}, data []byte) []interface{} {

	// defer func() {
	// 	if r := recover(); r != nil {
	// 		log.Println("RECOVERED", len(data), data)
	// 	}
	// }()

	msg, err := parse(data)
	recipientID := reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("RecipientID").String()

	if err != nil {
		log.Println("FUCKKKKK")
		// panic(err)
		res := struct {
			Type string
		}{}

		if err := json.Unmarshal(data, &res); err != nil {
			log.Println(res)
		}
	}

	// log.Println("Received message:", msg)
	// log.Println("notify parsed", msg, reflect.TypeOf(msg))

	replies := make([]interface{}, 0)

	for _, listener := range listeners {
		if listener.ServerID != "" && listener.ServerID != recipientID && !listener.Distributor {
			continue
		}

		reply := listener.Handle(c, msg)

		if reply == nil {
			continue
		}

		messageID := reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("MessageID").String()
		reflect.ValueOf(reply).Elem().FieldByName("Message").FieldByName("MessageID").Set(reflect.ValueOf(messageID))

		replies = append(replies, reply)
	}

	return replies
}
