package net

import "log"

func (n *Network) processMessage(client, msg interface{}) interface{} {
	log.Println("listener 3")
	c := client.(*Client)

	defer log.Println("listener 3 done")

	switch msg := msg.(type) {
	case PingMessage:
		c.ID = msg.Message.SenderID
		c.ClientRoutingInfo = ClientRoutingInfo{
			Distance: 1,
		}
		c.Neighbor = true

		log.Println("locking")
		n.Lock()
		n.clients[c.ID] = c
		n.Unlock()
		log.Println("locking done")

		return NewPongMessage(n.distributor)
	case RoutingMessage:
		n.UpdateRoutes(c, msg.Distances)
	}

	return nil
}
