package net

import (
	"encoding"
	"net"
	"sync"
)

// Actually can't be increased past this number -- kcp-go enforces a packet
// size limit of 1500 bytes, and 128 bytes are reserved for the header.
const maxBufferSize = 1372

type ClientRoutingInfo struct {
	// Distance to this client. Right now, this is just the number of nodes
	// packets need to travel through in order to reach this client. In the
	// future, we can consider replacing this with ping times.
	Distance float64

	// True if this client is a distributor.
	Distributor bool
}

type ConnectionState int

const (
	Disconnected ConnectionState = iota
	Connected
	TimedOut
)

type Client struct {
	sync.RWMutex

	Delegate ClientDelegate

	// The address to which to send messages in order to reach this client. If
	// this client is reached through a distributor, this address will be the
	// address of the distributor.
	Addr string

	ClientRoutingInfo

	// True if this client is directly connected to us, e.g. not through
	// another client or a distributor.
	Neighbor bool

	// should be set at the beginning and saved
	Username string

	// ID uniquely identifying the client.
	ID string

	// ID of the client through which this client is reached.
	NextHop string

	Seq int

	conn net.Conn

	sendCh chan []byte
	recvCh chan []byte

	State ConnectionState
}

// NewClient creates a client with the given address.
func NewNeighboringClient(addr string) *Client {
	return &Client{
		Addr:     addr,
		ID:       addr,
		Neighbor: true,
		sendCh:   make(chan []byte, maxBufferSize),
	}
}

func NewDistantClient(id, nextHop string, distance float64, distributor bool) *Client {
	return &Client{
		ID:      id,
		NextHop: nextHop,
		ClientRoutingInfo: ClientRoutingInfo{
			Distance:    distance,
			Distributor: distributor,
		},
	}
}

// start begins reading and writing messages with this client.
func (c *Client) start(conn net.Conn) {
	c.conn = conn

	go c.readPump()
	go c.writePump()
}

func (c *Client) disconnect() {
	c.Lock()
	if c.State != Connected {
		c.Unlock()
		return
	}

	c.State = Disconnected

	close(c.sendCh)
	close(c.recvCh)

	if c.conn != nil {
		c.conn.Close()
	}
	c.Unlock()

	c.Delegate.ClientDisconnected(c.ID)
}

// readPump pumps messages from the UDP connection to processMessage.
func (c *Client) readPump() {
	buf := make([]byte, maxBufferSize)

	for {
		n, err := c.conn.Read(buf)

		if err != nil {
			c.disconnect()
			return
		}

		data := make([]byte, n)
		copy(data, buf[:n])

		c.recvCh <- data

		// // Randomly drop packets if debugging
		// dropRate := arcade.Server.Network.GetDropRate()

		// if dropRate > 0 && rand.Float64() < dropRate {
		// 	continue
		// }

		// // Handle the message
		// arcade.Server.handleMessage(c, data)
	}
}

// writePump pumps messages from the sendCh to the client's UDP connection.
func (c *Client) writePump() {
	for {
		data, ok := <-c.sendCh

		if !ok {
			c.disconnect()
			return
		}

		_, err := c.conn.Write(data)

		if err != nil {
			c.disconnect()
			return
		}
	}
}

// send sends a message to the client.
func (c *Client) send(msg interface{}) {
	// Randomly drop packets if debugging
	// TODO: Fix
	// dropRate := arcade.Server.Network.GetDropRate()

	// if dropRate > 0 && rand.Float64() < dropRate {
	// 	return
	// }

	data, _ := msg.(encoding.BinaryMarshaler).MarshalBinary()
	c.sendCh <- data
}
