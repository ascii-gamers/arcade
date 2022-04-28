package arcade

import (
	"encoding"
	"fmt"
	"log"
	"net"
	"reflect"
	"sync"

	"github.com/google/uuid"
)

const maxBufferSize = 1024

type Client struct {
	// The address to which to send messages in order to reach this client. If
	// this client is reached through a distributor, this address will be the
	// address of the distributor.
	Addr string

	// ID uniquely identifying the client.
	ID string

	Distributor bool

	// True if this client is directly connected to us, e.g. not through
	// another client or a distributor.
	Neighbor bool

	conn    net.Conn
	connMux sync.Mutex

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

		data := make([]byte, n)
		copy(data, buf[:n])

		p, err := parseMessage(data)

		if err != nil {
			log.Fatal(err)
		}

		msg := reflect.ValueOf(p).FieldByName("Message").Interface().(Message)

		if distributor {
			fmt.Printf("Received '%s' from %s\n", msg.Type, msg.SenderID[:4])

			if msg.Type == "error" {
				fmt.Println(p)
			}
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

		switch p := p.(type) {
		case GetClientsMessage:
			res = NewClientsMessage(server.getClients())
		case PingMessage:
			c.ID = p.ID
			c.Neighbor = true

			server.Lock()
			server.clients[c.ID] = c
			server.Unlock()

			res = NewPongMessage(server.ID, server.getClients(), distributor)
		case PongMessage:
			c.ID = p.ID
			c.Distributor = p.Distributor
			c.Neighbor = true

			server.Lock()
			server.clients[c.ID] = c
			server.Unlock()

			server.AddClients(c, p.Clients)
			c.connectedCh <- true
		default:
			if msg.RecipientID != server.ID {
				if !distributor {
					panic("Uh oh")
				}

				fmt.Println("Forwarding message to", msg.RecipientID[:4])
				fmt.Println(p)

				server.RLock()
				recipient, ok := server.clients[msg.RecipientID]
				server.RUnlock()

				if ok {
					recipient.sendCh <- data
					continue
				} else {
					res = NewErrorMessage("Invalid recipient")
				}
			} else {
				if distributor {
					panic("Recipient: " + msg.RecipientID + ", self: " + server.ID)
				}

				res = processMessage(c, p)
			}
		}

		if res == nil {
			continue
		} else if err, ok := res.(error); ok {
			res = NewErrorMessage(err.Error())
		}

		// Set sender and recipient IDs
		reflect.ValueOf(res).Elem().FieldByName("Message").FieldByName("RecipientID").Set(reflect.ValueOf(msg.SenderID))
		reflect.ValueOf(res).Elem().FieldByName("Message").FieldByName("SenderID").Set(reflect.ValueOf(server.ID))

		// Set message ID if there was one in the sent packet
		reflect.ValueOf(res).Elem().FieldByName("Message").FieldByName("ID").Set(reflect.ValueOf(messageID))

		resData, err := res.(encoding.BinaryMarshaler).MarshalBinary()

		if err != nil {
			log.Fatal(err)
			return
		}

		c.sendCh <- resData
	}
}

// writePump pumps messages from the sendCh to the client's UDP connection.
func (c *Client) writePump(startedCh chan bool) {
	startedCh <- true

	for {
		_, err := c.conn.Write(<-c.sendCh)

		if err != nil {
			log.Fatal(err)
		}
	}
}

// send sends a message to the client.
func (c *Client) send(msg interface{}) {
	// Set sender ID
	reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("SenderID").Set(reflect.ValueOf(server.ID))

	// Set recipient ID
	reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("RecipientID").Set(reflect.ValueOf(c.ID))

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
