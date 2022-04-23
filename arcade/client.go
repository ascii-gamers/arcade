package arcade

import (
	"encoding"
	"log"
	"net"
)

type Client struct {
	Addr   string
	conn   net.Conn
	sendCh chan []byte
}

// NewClient creates a client with the given address.
func NewClient(addr string) *Client {
	return &Client{
		Addr:   addr,
		sendCh: make(chan []byte),
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

		res := processMessage(c, buf[:n])

		if res == nil {
			continue
		} else if err, ok := res.(error); ok {
			res = NewErrorMessage(err.Error())
		}

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
