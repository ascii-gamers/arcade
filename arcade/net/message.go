package net

func (n *Network) processMessage(client, msg interface{}) interface{} {
	c := client.(*Client)

	switch msg := msg.(type) {
	case *PingMessage:
		c.RLock()
		clientID := c.ID
		c.RUnlock()

		if value, ok := n.clients.Load(clientID); ok {
			existingClient := value.(*Client)

			existingClient.RLock()
			existingClientNextHop := existingClient.NextHop
			existingClient.RUnlock()

			if existingClient != c && existingClientNextHop != "" {
				existingClient.disconnect()
				n.clients.Delete(clientID)
			}
		}

		c.Lock()
		c.ID = msg.Message.SenderID
		c.ClientRoutingInfo = ClientRoutingInfo{
			Distributor: msg.Distributor,
			Distance:    1,
		}
		c.Neighbor = true
		c.Unlock()

		n.clients.Store(msg.Message.SenderID, c)

		return NewPongMessage(n.distributor)
	case *RoutingMessage:
		n.UpdateRoutes(c, msg.Distances)
	}

	return nil
}
