package net

import (
	"arcade/arcade/message"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xtaci/kcp-go/v5"
)

type Network struct {
	sync.RWMutex

	clients     map[string]*Client
	distributor bool
	dropRate    float64
	me          string
	port        int

	pendingMessagesMux sync.RWMutex
	pendingMessages    map[string]chan interface{}
}

const timeoutInterval = time.Second
const sendAndReceiveTimeout = time.Second

func NewNetwork(me string, port int) *Network {
	message.Register(PingMessage{Message: message.Message{Type: "ping"}})
	message.Register(PongMessage{Message: message.Message{Type: "pong"}})
	message.Register(RoutingMessage{Message: message.Message{Type: "routing"}})

	n := &Network{
		clients:         make(map[string]*Client),
		me:              me,
		port:            port,
		pendingMessages: make(map[string]chan interface{}),
	}

	message.AddListener(n.processMessage)
	return n
}

func (n *Network) Addr() string {
	ip, _ := GetLocalIP()
	return fmt.Sprintf("%s:%d", ip, n.port)
}

func (n *Network) Connect(addr string, conn net.Conn) (*Client, error) {
	n.Lock()
	defer n.Unlock()

	// TODO: Handle this
	for _, client := range n.clients {
		if client.Addr == addr {
			return nil, errors.New("already connected to " + addr)
		}
	}

	c := &Client{
		Addr:      addr,
		Neighbor:  true,
		recvCh:    make(chan []byte, maxBufferSize),
		sendCh:    make(chan []byte, maxBufferSize),
		pingTimes: make(map[string]time.Time),
	}

	// log.Println("ME: ", c.ID)
	go func() {
		for {
			data, ok := <-c.recvCh

			if !ok {
				break
			}

			for _, reply := range message.Notify(c, data) {
				// log.Println("REPLY: ", reply, c.ID)
				n.Send(c, reply)
			}
		}
	}()

	if conn == nil {
		var err error
		conn, err = kcp.Dial(c.Addr)

		if err != nil {
			return nil, err
		}
	}

	c.start(conn)

	c.connectedCh = make(chan bool)

	msg := NewPingMessage()
	msg.Message.MessageID = uuid.NewString()

	c.Lock()
	c.pingTimes[msg.Message.MessageID] = time.Now()
	c.Unlock()

	n.Send(c, msg)

	// Timeout if we don't receive a response
	time.AfterFunc(timeoutInterval, func() {
		c.Lock()
		defer c.Unlock()

		if c.State != Connected {
			c.State = TimedOut
			c.connectedCh <- false
		}
	})

	if !<-c.connectedCh {
		return nil, errors.New("client timed out")
	}

	go n.PropagateRoutes()

	return c, nil
}

func (n *Network) Disconnect(id string) {
	c, ok := n.GetClient(id)

	if !ok {
		return
	}

	c.Lock()
	defer c.Unlock()

	close(c.sendCh)
	close(c.recvCh)
	close(c.connectedCh)

	if c.conn != nil {
		c.conn.Close()
	}
}

func (n *Network) GetClient(id string) (*Client, bool) {
	n.RLock()
	defer n.RUnlock()

	c, ok := n.clients[id]
	return c, ok
}

func (n *Network) ClientsRange(f func(*Client) bool) {
	n.RLock()
	defer n.RUnlock()

	for _, client := range n.clients {
		if !f(client) {
			break
		}
	}
}

func (n *Network) Send(client *Client, msg interface{}) bool {
	// Set sender and recipient IDs
	// fmt.Println("AFTER1.5", msg, reflect.TypeOf(msg).Kind())

	// log.Println("in Send: ", msg)
	reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("SenderID").Set(reflect.ValueOf(n.me))
	reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("RecipientID").Set(reflect.ValueOf(client.ID))

	if client.NextHop == "" {
		client.send(msg)
		return true
	}

	n.RLock()
	defer n.RUnlock()

	servicer, ok := n.clients[client.NextHop]

	if !ok {
		return false
	}

	servicer.send(msg)
	return true
}

func (n *Network) SendAndReceive(client *Client, msg interface{}) (interface{}, error) {

	// log.Println("in SendAndReceive: ", msg)
	// Set message ID
	messageID := uuid.NewString()
	reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("MessageID").Set(reflect.ValueOf(messageID))

	// Set up receive chan
	recvCh := make(chan interface{}, 1)

	n.pendingMessagesMux.Lock()
	n.pendingMessages[messageID] = recvCh
	n.pendingMessagesMux.Unlock()

	// Send message
	n.Send(client, msg)

	time.AfterFunc(sendAndReceiveTimeout, func() {
		n.pendingMessagesMux.Lock()
		if _, ok := n.pendingMessages[messageID]; ok {
			delete(n.pendingMessages, messageID)
			close(recvCh)
		}
		n.pendingMessagesMux.Unlock()
	})

	// Wait for response
	recvMsg, ok := <-recvCh

	if !ok {
		return nil, fmt.Errorf("Timed out")
	}

	n.pendingMessagesMux.Lock()
	delete(n.pendingMessages, messageID)
	n.pendingMessagesMux.Unlock()

	return recvMsg, nil
}

func (n *Network) SignalReceived(messageID string, resp interface{}) {

	n.pendingMessagesMux.RLock()
	defer n.pendingMessagesMux.RUnlock()
	log.Println("SIGNAL RECEIVED", n.pendingMessages)
	if ch, ok := n.pendingMessages[messageID]; ok {
		log.Println("SIGNAL RECEIVED", "found message")
		ch <- resp
		close(ch)
	}
}

func (n *Network) SendNeighbors(msg interface{}) {
	n.RLock()
	defer n.RUnlock()

	// Set sender ID
	reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("SenderID").Set(reflect.ValueOf(n.me))

	for _, client := range n.clients {
		if !client.Neighbor {
			continue
		}

		// Set recipient ID
		reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("RecipientID").Set(reflect.ValueOf(client.ID))

		client.send(msg)
	}
}

func (n *Network) getDistanceVector() map[string]*ClientRoutingInfo {
	distances := make(map[string]*ClientRoutingInfo)

	for clientID, client := range n.clients {
		if !client.Neighbor {
			continue
		}

		distances[clientID] = &client.ClientRoutingInfo
	}

	return distances
}

func (n *Network) PropagateRoutes() {
	n.RLock()
	distances := n.getDistanceVector()
	n.RUnlock()

	n.SendNeighbors(NewRoutingMessage(distances))
}

func (n *Network) UpdateRoutes(from *Client, routingTable map[string]*ClientRoutingInfo) {
	n.Lock()
	defer n.Unlock()

	changes := 0

	for clientID, client := range n.clients {
		delete(routingTable, clientID)

		if clientID == from.ID {
			continue
		}

		// Bellman-Ford equation: Update least-cost paths to all other clients
		if c, ok := routingTable[clientID]; ok && c.Distance < client.Distance {
			fmt.Println("new path to", clientID, "cost=", c.Distance)

			client.Lock()
			client.Distance = c.Distance

			from.RLock()
			client.NextHop = from.ID
			client.conn = from.conn
			from.RUnlock()

			client.Unlock()
			changes++
		}
	}

	for clientID, c := range routingTable {
		if clientID == n.me {
			continue
		}

		n.clients[clientID] = NewDistantClient(clientID, from.ID, c.Distance+1, c.Distributor)
		changes++
	}

	if changes == 0 {
		return
	}

	go n.PropagateRoutes()
}

func (n *Network) GetDropRate() float64 {
	n.RLock()
	defer n.RUnlock()

	// Dropping is applied to sending and receiving, so do some math to get the
	// correct rate. X + X * (1 - X) = Y, solve for Y
	return 1 - math.Sqrt(1-n.dropRate)
}

func (n *Network) SetDropRate(rate float64) {
	n.Lock()
	defer n.Unlock()

	n.dropRate = rate
}
