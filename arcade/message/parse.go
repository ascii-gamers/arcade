package message

import (
	"encoding/json"
	"errors"
	"reflect"
)

var types = map[string]interface{}{}

func Register(msg interface{}) {
	if reflect.TypeOf(msg).Kind() == reflect.Pointer {
		panic("msg must be a value")
	}

	messageType := reflect.ValueOf(msg).FieldByName("Message").FieldByName("Type").String()

	if _, ok := types[messageType]; ok {
		return
	}

	types[messageType] = msg
}

func parse(data []byte) (interface{}, error) {
	res := struct {
		Type string
	}{}

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	for messageType := range types {
		if messageType != res.Type {
			continue
		}

		p := reflect.New(reflect.TypeOf(types[messageType])).Interface()

		if err := json.Unmarshal(data, p); err != nil {
			panic(err)
		}

		return reflect.ValueOf(p).Elem().Interface(), nil
	}

	return nil, errors.New("unknown message type '" + res.Type + "'")
}
