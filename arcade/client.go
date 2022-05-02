package arcade

import (
	"encoding"
	"log"
	"net"
	"reflect"
	"sync"

	"github.com/google/uuid"
)

const maxBufferSize = 1024

type ClientRoutingInfo struct {
	Distributor bool
	Distance    float64
}

type Client struct {
	sync.RWMutex

	// The address to which to send messages in order to reach this client. If
	// this client is reached through a distributor, this address will be the
	// address of the distributor.
	Addr string

	// True if we're connected to this client.
	Connected bool

	// Distance to this client. Right now, this is just the number of nodes
	// packets need to travel through in order to reach this client. In the
	// future, we can consider replacing this with ping times.
	// Distance float64

	// True if this client is a distributor.
	// Distributor bool

	ClientRoutingInfo

	// True if this client is directly connected to us, e.g. not through
	// another client or a distributor.
	Neighbor bool

	// should be set at the beginning and saved
	Username string

	// ID uniquely identifying the client.
	ID string

	// ID of the client through which this client is reached.
	ServicerID string

	conn net.Conn

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

func (c *Client) connect() {
	c.Connected = true
}

// start begins reading and writing messages with this client.
func (c *Client) start(conn net.Conn) {
	c.conn = conn

	go c.readPump()
	go c.writePump()

	// TODO: Think about a better solution
	// time.Sleep(100 * time.Millisecond)
}

// readPump pumps messages from the UDP connection to processMessage.
func (c *Client) readPump() {
	buf := make([]byte, maxBufferSize)

	for {
		n, err := c.conn.Read(buf)

		if err != nil {
			log.Fatal(err)
		}

		data := make([]byte, n)
		copy(data, buf[:n])

		arcade.Server.handleMessage(c, data)
	}
}

// writePump pumps messages from the sendCh to the client's UDP connection.
func (c *Client) writePump() {
	for {
		data := <-c.sendCh

		c.RLock()
		_, err := c.conn.Write(data)
		c.RUnlock()

		if err != nil {
			log.Fatal(err)
		}
	}
}

// send sends a message to the client.
func (c *Client) send(msg interface{}) {
	data, _ := msg.(encoding.BinaryMarshaler).MarshalBinary()
	c.sendCh <- data
}

func (c *Client) SendAndReceive(msg interface{}) interface{} {
	// Set message ID
	messageID := uuid.NewString()
	reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("MessageID").Set(reflect.ValueOf(messageID))

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
