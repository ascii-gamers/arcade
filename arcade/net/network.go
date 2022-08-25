package net

import (
	"arcade/arcade/message"
	"encoding/json"
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

	Delegate NetworkDelegate

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

func NewNetwork(me string, port int, distributor bool) *Network {
	message.Register(PingMessage{Message: message.Message{Type: "ping"}})
	message.Register(PongMessage{Message: message.Message{Type: "pong"}})
	message.Register(RoutingMessage{Message: message.Message{Type: "routing"}})

	n := &Network{
		clients:         make(map[string]*Client),
		me:              me,
		port:            port,
		distributor:     distributor,
		pendingMessages: make(map[string]chan interface{}),
	}

	message.AddListener(message.Listener{
		Distributor: true,
		ServerID:    n.me,
		Handle:      n.processMessage,
	})

	return n
}

func (n *Network) Addr() string {
	ip, _ := GetLocalIP()
	return fmt.Sprintf("%s:%d", ip, n.port)
}

func (n *Network) Connect(addr, id string, conn net.Conn) (*Client, error) {
	c, ok := n.GetClient(id)

	if !ok || c.State == Disconnected || c.NextHop != "" {
		c = &Client{
			Delegate: n,
			Addr:     addr,
			ID:       id,
			Neighbor: true,
			State:    Connecting,
			recvCh:   make(chan []byte, maxBufferSize),
			sendCh:   make(chan []byte, maxBufferSize),
		}

		go func() {
			for {
				data, ok := <-c.recvCh

				if !ok {
					break
				}

				// Get sender ID
				res := struct {
					SenderID string
				}{}

				if err := json.Unmarshal(data, &res); err != nil {
					break
				}

				sender, ok := n.GetClient(res.SenderID)

				if !ok {
					sender = c
				}

				for _, reply := range message.Notify(c, data) {
					n.Send(sender, reply)
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
	}

	// Send ping and wait for reply
	start := time.Now()
	res, err := n.SendAndReceive(c, NewPingMessage(n.distributor))
	end := time.Now()

	p, ok := res.(*PongMessage)

	if !ok || err != nil {
		c.Lock()
		c.State = TimedOut
		c.Unlock()

		return nil, errors.New("timed out")
	}

	clientID := p.SenderID

	c.Lock()
	c.Distributor = p.Distributor
	c.ID = clientID
	c.ClientRoutingInfo = ClientRoutingInfo{
		Distance:    float64(end.Sub(start)),
		Distributor: p.Distributor,
	}
	c.Neighbor = true
	c.State = Connected
	c.Unlock()

	n.Lock()
	n.clients[clientID] = c
	n.Unlock()

	if !p.Distributor && n.Delegate != nil {
		n.Delegate.ClientConnected(clientID)
	}

	go n.PropagateRoutes()

	return c, nil
}

func (n *Network) Disconnect(id string) {
	c, ok := n.GetClient(id)

	if !ok {
		return
	}

	c.disconnect()
}

func (n *Network) GetClient(id string) (*Client, bool) {
	n.RLock()
	defer n.RUnlock()

	c, ok := n.clients[id]
	return c, ok
}

func (n *Network) ClientsRange(f func(*Client) bool) {
	n.RLock()
	clients := n.clients
	n.RUnlock()

	for _, client := range clients {
		if !f(client) {
			break
		}
	}
}

func (n *Network) Send(client *Client, msg interface{}) bool {
	// Set sender and recipient IDs
	reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("SenderID").Set(reflect.ValueOf(n.me))
	reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("RecipientID").Set(reflect.ValueOf(client.ID))

	if client.NextHop == "" {
		client.Send(msg)
		return true
	}

	n.RLock()
	defer n.RUnlock()

	servicer, ok := n.clients[client.NextHop]

	if !ok {
		return false
	}

	servicer.Send(msg)
	return true
}

func (n *Network) SendAndReceive(client *Client, msg interface{}) (interface{}, error) {
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
		return nil, fmt.Errorf("timed out")
	}

	n.pendingMessagesMux.Lock()
	delete(n.pendingMessages, messageID)
	n.pendingMessagesMux.Unlock()

	return recvMsg, nil
}

func (n *Network) SignalReceived(messageID string, resp interface{}) {
	n.pendingMessagesMux.RLock()
	defer n.pendingMessagesMux.RUnlock()

	if ch, ok := n.pendingMessages[messageID]; ok {
		ch <- resp
		close(ch)
	}
}

func (n *Network) SendNeighbors(msg interface{}) {
	n.RLock()
	clients := n.clients
	n.RUnlock()

	// Set sender ID
	reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("SenderID").Set(reflect.ValueOf(n.me))

	for _, client := range clients {
		if !client.Neighbor || (client.State != Connected && client.State != Connecting) {
			continue
		}

		// Set recipient ID
		reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("RecipientID").Set(reflect.ValueOf(client.ID))

		client.Send(msg)
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
			log.Println("New path to", clientID, "cost=", c.Distance)

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

		n.clients[clientID] = &Client{
			ID:      clientID,
			NextHop: from.ID,
			ClientRoutingInfo: ClientRoutingInfo{
				Distance:    c.Distance + 1,
				Distributor: c.Distributor,
			},
			State: Connected,
		}

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

//
// ClientDelegate methods
//

func (n *Network) ClientDisconnected(clientID string) {
	n.Lock()
	delete(n.clients, clientID)
	n.Unlock()

	if n.Delegate != nil {
		n.Delegate.ClientDisconnected(clientID)
	}
}
