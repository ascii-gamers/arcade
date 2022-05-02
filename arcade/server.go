package arcade

import (
	"encoding"
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"

	"github.com/google/uuid"
	"github.com/xtaci/kcp-go/v5"
)

type Server struct {
	sync.RWMutex

	Network *Network

	Addr string
	ID   string
}

// NewServer creates the server with a given address.
func NewServer(addr string) *Server {
	serverID := uuid.NewString()

	return &Server{
		Addr:    addr,
		Network: NewNetwork(serverID),
		ID:      serverID,
	}
}

// connect connects to a client at a given address.
func (s *Server) connect(c *Client) error {
	sess, err := kcp.Dial(c.Addr)

	if err != nil {
		return err
	}

	c.start(sess)

	c.connectedCh = make(chan bool)
	s.Network.Send(c, NewPingMessage(s.ID))

	// TODO: timeout if no response
	if !<-c.connectedCh {
		return errors.New("client failed to connect")
	}

	go s.Network.PropagateRoutes()

	return nil
}

func (s *Server) handleMessage(c *Client, data []byte) {
	p, err := parseMessage(data)

	if err != nil {
		log.Fatal(err)
	}

	msg := reflect.ValueOf(p).FieldByName("Message").Interface().(Message)

	if arcade.Distributor {
		fmt.Println(msg)
		fmt.Printf("Received '%s' from %s\n", msg.Type, msg.SenderID[:4])

		if msg.Type == "error" {
			fmt.Println(p)
		}
	}

	// Get message ID and send signal if we're waiting on this one
	messageID := reflect.ValueOf(p).FieldByName("Message").FieldByName("MessageID").String()

	c.pendingMessagesMux.RLock()
	recvCh, ok := c.pendingMessages[messageID]
	c.pendingMessagesMux.RUnlock()

	if ok {
		recvCh <- p
	}

	// Process message and prepare response
	var res interface{}

	switch p := p.(type) {
	// case ClientsMessage:
	// 	pendingActions := make([]func(), 0)

	// 	// Find all clients we're connected to through this client
	// 	existingClients := make(map[string]bool)

	// 	for clientID, client := range s.clients {
	// 		if client.Connected && client.ServicerID == c.ID {
	// 			existingClients[clientID] = true
	// 		}
	// 	}

	// 	for clientID, distance := range p.Clients {
	// 		delete(existingClients, clientID)

	// 		if clientID == s.ID {
	// 			continue
	// 		}

	// 		pendingActions = append(pendingActions, func() {
	// 			s.AddClient(&Client{
	// 				Addr:            c.Addr,
	// 				Distance:        distance + 1,
	// 				ID:              clientID,
	// 				ServicerID:      c.ID,
	// 				sendCh:          c.sendCh,
	// 				pendingMessages: make(map[string]chan interface{}),
	// 			})
	// 		})
	// 	}

	// 	for clientID := range existingClients {
	// 		mgr.ProcessEvent(&ClientDisconnectEvent{
	// 			ClientID: clientID,
	// 		})

	// 		pendingActions = append(pendingActions, func() {
	// 			delete(s.clients, clientID)
	// 		})
	// 	}

	// 	s.Lock()
	// 	for _, action := range pendingActions {
	// 		action()
	// 	}
	// 	s.Unlock()
	case DisconnectMessage:
		arcade.ViewManager.ProcessEvent(&ClientDisconnectEvent{
			ClientID: c.ID,
		})

		s.Network.DeleteClient(c.ID)
	// case GetClientsMessage:
	// 	res = NewClientsMessage(s.getClients())
	case PingMessage:
		c.ID = p.ID
		c.ClientRoutingInfo = ClientRoutingInfo{
			Distance: 1,
		}
		c.Neighbor = true

		s.Network.AddClient(c)

		res = NewPongMessage(s.ID, arcade.Distributor)
	case PongMessage:
		c.ID = p.ID
		c.ClientRoutingInfo = ClientRoutingInfo{
			Distance:    1,
			Distributor: p.Distributor,
		}
		c.Neighbor = true

		s.Network.AddClient(c)

		c.connectedCh <- true
	case RoutingMessage:
		s.Network.UpdateRoutes(c, p.Distances)
	default:
		if msg.RecipientID != s.ID {
			if !arcade.Distributor {
				fmt.Println(p)
				panic(fmt.Sprintf("Recipient ID is %s, but server ID is %s", msg.RecipientID, s.ID))
			}

			fmt.Println("Forwarding message to", msg.RecipientID[:4])
			fmt.Println(p)

			s.RLock()
			recipient, ok := s.Network.GetClient(msg.RecipientID)
			s.RUnlock()

			if ok {
				recipient.sendCh <- data
				return
			} else {
				res = NewErrorMessage("Invalid recipient")
			}
		} else {
			if arcade.Distributor {
				fmt.Println(p)
				panic("Recipient: " + msg.RecipientID + ", self: " + s.ID)
			}

			res = processMessage(c, p)
		}
	}

	if res == nil {
		return
	} else if err, ok := res.(error); ok {
		res = NewErrorMessage(err.Error())
	}

	// Set sender and recipient IDs
	reflect.ValueOf(res).Elem().FieldByName("Message").FieldByName("RecipientID").Set(reflect.ValueOf(msg.SenderID))
	reflect.ValueOf(res).Elem().FieldByName("Message").FieldByName("SenderID").Set(reflect.ValueOf(s.ID))

	// Set message ID if there was one in the sent packet
	reflect.ValueOf(res).Elem().FieldByName("Message").FieldByName("MessageID").Set(reflect.ValueOf(messageID))

	resData, err := res.(encoding.BinaryMarshaler).MarshalBinary()

	if err != nil {
		log.Fatal(err)
		return
	}

	c.sendCh <- resData
}

func (s *Server) startWithNextOpenPort() {
	for {
		s.Addr = fmt.Sprintf("127.0.0.1:%d", arcade.Port)
		s.start()

		arcade.Port++
	}
}

// startServer starts listening for connections on a given address.
func (s *Server) start() error {
	listener, err := kcp.Listen(s.Addr)

	if err != nil {
		return err
	}

	fmt.Printf("Listening at %s...\n", s.Addr)
	fmt.Printf("ID: %s\n", s.ID)

	for {
		// Wait for new client connections
		s, err := listener.Accept()

		if err != nil {
			log.Fatal(err)
		}

		client := &Client{
			Addr:   s.RemoteAddr().String(),
			sendCh: make(chan []byte),
		}

		client.start(s)
	}
}

// func (s *Server) getClients() map[string]float64 {
// 	s.RLock()
// 	defer s.RUnlock()

// 	clients := s.clients
// 	clientDists := make(map[string]float64, len(clients))

// 	for i := range clients {
// 		if clients[i].Distributor {
// 			continue
// 		}

// 		clientDists[clients[i].ID] = clients[i].Distance
// 	}

// 	return clientDists
// }
