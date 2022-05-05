package arcade

import (
	"fmt"
	"reflect"
	"sync"
)

type Network struct {
	sync.RWMutex

	clients  map[string]*Client
	dropRate float64
	me       string
}

func NewNetwork(me string) *Network {
	return &Network{
		clients: make(map[string]*Client),
		me:      me,
	}
}

func (n *Network) GetClient(id string) (*Client, bool) {
	n.RLock()
	defer n.RUnlock()

	c, ok := n.clients[id]
	return c, ok
}

func (n *Network) AddClient(c *Client) {
	n.Lock()
	defer n.Unlock()

	n.clients[c.ID] = c
	go n.PropagateRoutes()
}

func (n *Network) DeleteClient(id string) {
	n.Lock()
	defer n.Unlock()

	delete(n.clients, id)
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
			// TODO: Fix this
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

	return n.dropRate
}

func (n *Network) SetDropRate(rate float64) {
	n.Lock()
	defer n.Unlock()

	n.dropRate = rate
}
