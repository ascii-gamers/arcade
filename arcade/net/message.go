package net

import (
	"log"
	"time"
)

func (n *Network) processMessage(client, msg interface{}) interface{} {
	c := client.(*Client)
	log.Println("GOT IT")

	switch msg := msg.(type) {
	case PingMessage:
		c.ID = msg.Message.SenderID
		c.ClientRoutingInfo = ClientRoutingInfo{
			Distance: 1,
		}
		c.Neighbor = true

		n.Lock()
		n.clients[c.ID] = c
		n.Unlock()

		log.Println("replying with pong")
		return NewPongMessage(n.distributor)
	case PongMessage:
		c.Lock()
		pingTime := time.Since(c.pingTimes[msg.Message.MessageID])
		delete(c.pingTimes, msg.Message.MessageID)
		c.Unlock()

		c.Lock()
		c.ID = msg.Message.SenderID
		c.ClientRoutingInfo = ClientRoutingInfo{
			Distance:    float64(pingTime.Milliseconds()),
			Distributor: n.distributor,
		}
		c.Neighbor = true

		if c.State == Disconnected {
			n.Lock()
			n.clients[c.ID] = c
			n.Unlock()

			c.State = Connected
			c.connectedCh <- true

			// TODO: Reimplement
			// if !c.Distributor {
			// 	arcade.ViewManager.ProcessEvent(NewClientConnectEvent(c.ID))
			// }
		}
		c.Unlock()
	case RoutingMessage:
		n.UpdateRoutes(c, msg.Distances)
	}

	return nil
}
