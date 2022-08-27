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

	clients     sync.Map
	distributor bool
	dropRate    float64
	me          string
	port        int

	pendingMessagesMux sync.RWMutex
	pendingMessages    map[string]chan interface{}
}

const maxTimeoutRetries = 1
const sendAndReceiveTimeout = time.Second

func NewNetwork(me string, port int, distributor bool) *Network {
	message.Register(PingMessage{Message: message.Message{Type: "ping"}})
	message.Register(PongMessage{Message: message.Message{Type: "pong"}})
	message.Register(RoutingMessage{Message: message.Message{Type: "routing"}})

	n := &Network{
		clients:         sync.Map{},
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
	var c *Client

	if id != "" {
		// Find existing client by ID
		c, _ = n.GetClient(id)
	} else {
		// Find existing client by IP address
		// TODO: optimize
		n.clients.Range(func(key, value any) bool {
			client := value.(*Client)

			if client.Addr == addr {
				c = client
				return false
			}

			return true
		})
	}

	if c != nil {
		c.RLock()
		existingClientID := c.ID
		existingClientNextHop := c.NextHop
		existingClientState := c.State
		c.RUnlock()

		switch existingClientState {
		case Connecting:
			// Not too sure what to do here
			return nil, errors.New("already connecting")
		case Connected:
			if existingClientNextHop != "" {
				// If this client was previously discovered through another client, discard
				c.disconnect()
				n.clients.Delete(existingClientID)
			} else if err := n.ConnectClient(c, false); err != nil {
				// Try reconnecting
				c.disconnect()
				n.clients.Delete(existingClientID)
			} else {
				// We're good
				return c, nil
			}
		case Disconnected, TimedOut:
			c.disconnect()
			n.clients.Delete(existingClientID)
		}
	}

	c = &Client{
		Delegate: n,
		Addr:     addr,
		ID:       id,
		Neighbor: true,
		State:    Connecting,
	}

	if conn == nil {
		var err error
		conn, err = kcp.Dial(c.Addr)

		if err != nil {
			return nil, err
		}
	}

	c.start(conn)
	go n.handleMessages(c)

	return c, n.ConnectClient(c, true)
}

func (n *Network) ConnectClient(c *Client, retry bool) error {
	// Send ping and wait for reply
	start := time.Now()
	res, err := n.SendAndReceive(c, NewPingMessage(n.distributor))
	end := time.Now()

	p, ok := res.(*PongMessage)

	if !ok || err != nil {
		c.Lock()
		if retry && c.TimeoutRetries < maxTimeoutRetries {
			c.TimeoutRetries++
			c.Unlock()

			return n.ConnectClient(c, retry)
		}
		c.disconnect()
		n.clients.Delete(c.ID)
		c.State = TimedOut
		c.Unlock()

		return errors.New("timed out")
	}

	clientID := p.SenderID

	if value, ok := n.clients.Load(clientID); ok {
		existingClient := value.(*Client)

		if existingClient != c {
			existingClient.disconnect()
			n.clients.Delete(clientID)
		}
	}

	c.Lock()
	c.ID = clientID
	c.ClientRoutingInfo = ClientRoutingInfo{
		Distance:    float64(end.Sub(start).Milliseconds()),
		Distributor: p.Distributor,
	}
	c.Neighbor = true
	c.State = Connected
	c.TimeoutRetries = 0
	c.Unlock()

	n.clients.Store(clientID, c)

	if !p.Distributor && n.Delegate != nil {
		n.Delegate.ClientConnected(clientID)
	}

	go n.PropagateRoutes()

	return nil
}

func (n *Network) Disconnect(id string) {
	c, ok := n.GetClient(id)

	if !ok {
		return
	}

	c.disconnect()
}

func (n *Network) GetClient(id string) (*Client, bool) {
	c, ok := n.clients.Load(id)

	if !ok {
		return nil, false
	}

	return c.(*Client), true
}

func (n *Network) ClientsRange(f func(*Client) bool) {
	n.clients.Range(func(key, value interface{}) bool {
		return f(value.(*Client))
	})
}

func (n *Network) SendRaw(client *Client, msg interface{}) bool {
	client.RLock()
	if client.NextHop == "" {
		client.RUnlock()
		client.Send(msg)
		return true
	}
	client.RUnlock()

	n.RLock()
	defer n.RUnlock()

	servicer, ok := n.clients.Load(client.NextHop)

	if !ok {
		return false
	}

	servicer.(*Client).Send(msg)
	return true
}

func (n *Network) Send(client *Client, msg interface{}) bool {
	client.RLock()
	if client.State == Disconnected || client.State == TimedOut {
		client.RUnlock()
		return false
	}

	// Set sender and recipient IDs
	reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("SenderID").Set(reflect.ValueOf(n.me))
	reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("RecipientID").Set(reflect.ValueOf(client.ID))
	client.RUnlock()

	return n.SendRaw(client, msg)
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
	ok := n.Send(client, msg)

	if !ok {
		return nil, errors.New("send failed")
	}

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

	return recvMsg, nil
}

func (n *Network) SignalReceived(messageID string, resp interface{}) {
	n.pendingMessagesMux.Lock()
	defer n.pendingMessagesMux.Unlock()

	if ch, ok := n.pendingMessages[messageID]; ok {
		ch <- resp
		close(ch)
		delete(n.pendingMessages, messageID)
	}
}

func (n *Network) SendNeighbors(msg interface{}) {
	// Set sender ID
	reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("SenderID").Set(reflect.ValueOf(n.me))

	n.clients.Range(func(_, value any) bool {
		client := value.(*Client)
		client.RLock()

		if !client.Neighbor || (client.State != Connected && client.State != Connecting) {
			client.RUnlock()
			return true
		}

		// Set recipient ID
		reflect.ValueOf(msg).Elem().FieldByName("Message").FieldByName("RecipientID").Set(reflect.ValueOf(client.ID))

		client.RUnlock()
		client.Send(msg)
		return true
	})
}

func (n *Network) getDistanceVector() map[string]ClientRoutingInfo {
	distances := make(map[string]ClientRoutingInfo)

	n.clients.Range(func(key, value any) bool {
		clientID := key.(string)
		client := value.(*Client)
		distances[clientID] = client.ClientRoutingInfo
		return true
	})

	return distances
}

func (n *Network) PropagateRoutes() {
	n.RLock()
	distances := n.getDistanceVector()
	n.RUnlock()

	n.SendNeighbors(NewRoutingMessage(distances))
}

func (n *Network) UpdateRoutes(from *Client, routingTable map[string]ClientRoutingInfo) {
	n.Lock()
	defer n.Unlock()

	changes := 0

	n.clients.Range(func(key, value any) bool {
		clientID := key.(string)
		client := value.(*Client)

		delete(routingTable, clientID)

		if clientID == from.ID {
			return true
		}

		// Bellman-Ford equation: Update least-cost paths to all other clients
		if c, ok := routingTable[clientID]; ok && c.Distance < client.Distance && client.NextHop != "" {
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

		return true
	})

	for clientID, c := range routingTable {
		if clientID == n.me {
			continue
		}

		n.clients.Store(clientID, &Client{
			ID:       clientID,
			Delegate: n,
			NextHop:  from.ID,
			ClientRoutingInfo: ClientRoutingInfo{
				Distance:    c.Distance + 1,
				Distributor: c.Distributor,
			},
			State: Connected,
		})

		changes++
	}

	if changes == 0 {
		return
	}

	go n.PropagateRoutes()
}

func (n *Network) handleMessages(c *Client) {
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
	n.clients.Delete(clientID)

	if n.Delegate != nil {
		n.Delegate.ClientDisconnected(clientID)
	}
}
