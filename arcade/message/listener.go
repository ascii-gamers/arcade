package message

import (
	"encoding/json"
	"log"
	"reflect"
)

var listeners = make([]func(c, data interface{}) interface{}, 0)

func AddListener(listener func(c, data interface{}) interface{}) {
	listeners = append(listeners, listener)
}

func Notify(c interface{}, data []byte) []interface{} {

	// defer func() {
	// 	if r := recover(); r != nil {
	// 		log.Println("RECOVERED", len(data), data)
	// 	}
	// }()

	msg, err := parse(data)

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
