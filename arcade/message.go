package arcade

import (
	"encoding/json"
	"fmt"
)

type Message struct {
	Type string `json:"type"`
}

func processMessage(from *Client, data []byte) interface{} {
	p, err := parseMessage(data)

	if err != nil {
		return err
	}

	switch p := p.(type) {
	case ErrorMessage:
		return ErrorMessageHandler(from, p)
	case HelloMessage:
		return HelloMessageHandler(from, p)
	}

	return nil
}

func parseMessage(data []byte) (interface{}, error) {
	res := struct {
		Type string `json:"type"`
	}{}

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	switch res.Type {
	case "error":
		p := ErrorMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	case "hello":
		p := HelloMessage{}
		json.Unmarshal(data, &p)
		return p, nil
	default:
		return nil, fmt.Errorf("Unknown message type '%s'", res.Type)
	}
}
