package arcade

import (
	"encoding"
	"log"
	"net"
	"reflect"
	"sync"

	"github.com/google/uuid"
)

type Client struct {
	Addr   string
	conn   net.Conn
	sendCh chan []byte

	connectedCh chan bool

	pendingMessagesMux sync.RWMutex
	pendingMessages    map[string]chan interface{}
}

// NewClient creates a client with the given address.
func NewClient(addr string) *Client {
	return &Client{
		Addr:            addr,
		sendCh:          make(chan []byte),
		pendingMessages: make(map[string]chan interface{}),
	}
}

// start begins reading and writing messages with this client.
func (c *Client) start(conn net.Conn) {
	c.conn = conn

	// Ensure the read and write pumps start before returning
	readPumpStartedCh := make(chan bool)
	writePumpStartedCh := make(chan bool)

	go c.readPump(readPumpStartedCh)
	go c.writePump(writePumpStartedCh)

	<-readPumpStartedCh
	<-writePumpStartedCh
}

// readPump pumps messages from the UDP connection to processMessage.
func (c *Client) readPump(startedCh chan bool) {
	buf := make([]byte, maxBufferSize)
	startedCh <- true

	for {
		n, err := c.conn.Read(buf)

		if err != nil {
			log.Fatal(err)
		}

		p, err := parseMessage(buf[:n])

		if err != nil {
			log.Fatal(err)
		}

		// Get message ID and send signal if we're waiting on this one
		messageID := reflect.ValueOf(p).FieldByName("Message").FieldByName("ID").String()

		c.pendingMessagesMux.RLock()
		recvCh, ok := c.pendingMessages[messageID]
		c.pendingMessagesMux.RUnlock()

		if ok {
			recvCh <- p
		}

		// Process message and prepare response
		var res interface{}

		switch p.(type) {
		case PingMessage:
			res = NewPongMessage()
		case PongMessage:
			c.connectedCh <- true
		default:
			res = processMessage(c, p)
		}

		if res == nil {
			continue
		} else if err, ok := res.(error); ok {
			res = NewErrorMessage(err.Error())
		}

		// Set message ID if there was one in the sent packet
		reflect.ValueOf(res).Elem().FieldByName("Message").FieldByName("ID").Set(reflect.ValueOf(messageID))

		data, err := res.(encoding.BinaryMarshaler).MarshalBinary()

		if err != nil {
			log.Fatal(err)
			return
		}

		c.sendCh <- data
	}
}

// writePump pumps messages from the sendCh to the client's UDP connection.
func (c *Client) writePump(startedCh chan bool) {
	startedCh <- true

	for {
		if _, err := c.conn.Write(<-c.sendCh); err != nil {
			log.Fatal(err)
		}
	}
}

// send sends a message to the client.
func (c *Client) send(msg interface{}) {
	data, _ := msg.(encoding.BinaryMarshaler).MarshalBinary()
	c.sendCh <- data
}

func (c *Client) sendAndReceive(msg interface{}) interface{} {
	// Set message ID
	messageID := uuid.NewString()
	reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("ID").Set(reflect.ValueOf(messageID))

	// Set up receive chan
	recvCh := make(chan interface{}, 1)

	c.pendingMessagesMux.Lock()
	c.pendingMessages[messageID] = recvCh
	c.pendingMessagesMux.Unlock()

	// Send message
	data, _ := msg.(encoding.BinaryMarshaler).MarshalBinary()
	c.sendCh <- data

	// Wait for response
	recvMsg := <-recvCh

	c.pendingMessagesMux.Lock()
	delete(c.pendingMessages, messageID)
	c.pendingMessagesMux.Unlock()

	return recvMsg
}
