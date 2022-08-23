package message

import (
	"log"
	"reflect"
)

var listeners = make([]func(c, data interface{}) interface{}, 0)

func AddListener(listener func(c, data interface{}) interface{}) {
	listeners = append(listeners, listener)
}

func Notify(c interface{}, data []byte) []interface{} {
	msg, err := parse(data)

	if err != nil {
		panic(err)
	}

	log.Println("Received message:", msg)

	replies := make([]interface{}, 0)

	for _, listener := range listeners {
		reply := listener(c, msg)

		if reply == nil {
			continue
		}

		messageID := reflect.ValueOf(msg).FieldByName("Message").FieldByName("MessageID").String()
		reflect.ValueOf(reply).Elem().FieldByName("Message").FieldByName("MessageID").Set(reflect.ValueOf(messageID))

		replies = append(replies, reply)
	}

	return replies
}
