package message

import "log"

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
	// log.Println("notify parsed", msg, reflect.TypeOf(msg))

	replies := make([]interface{}, 0)

	for _, listener := range listeners {
		reply := listener(c, msg)

		if reply == nil {
			continue
		}

		replies = append(replies, reply)
	}

	return replies
}
