package arcade

import (
	"encoding"
	"errors"
	"fmt"
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
		panic(err)
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

	// Process message and prepare response
	var res interface{}

	switch p := p.(type) {
	case DisconnectMessage:
		arcade.ViewManager.ProcessEvent(&ClientDisconnectEvent{
			ClientID: c.ID,
		})

		s.Network.DeleteClient(c.ID)
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
		panic(err)
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
			panic(err)
		}

		client := NewNeighboringClient(s.RemoteAddr().String())
		client.start(s)
	}
}
