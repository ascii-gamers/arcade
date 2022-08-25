package net

func (n *Network) processMessage(client, msg interface{}) interface{} {
	c := client.(*Client)

	switch msg := msg.(type) {
	case *PingMessage:
		c.ID = msg.Message.SenderID
		c.ClientRoutingInfo = ClientRoutingInfo{
			Distributor: msg.Distributor,
			Distance:    1,
		}
		c.Neighbor = true

		n.Lock()
		n.clients[c.ID] = c
		n.Unlock()

		return NewPongMessage(n.distributor)
	case *RoutingMessage:
		n.UpdateRoutes(c, msg.Distances)
	}

	return nil
}
