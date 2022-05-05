package arcade

import (
	"encoding"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xtaci/kcp-go/v5"
)

const timeoutInterval = 2500 * time.Millisecond
const heartbeatInterval = time.Second

type ConnectedClientInfo struct {
	LastHeartbeat      time.Time
	HeartbeatSendTimes map[int]time.Time
	RTTs               []time.Duration
}

func (c *ConnectedClientInfo) GetMeanRTT() time.Duration {
	var sum time.Duration
	count := 0

	for i := len(c.RTTs) - 1; i >= 0 && i >= len(c.RTTs)-6; i-- {
		sum += c.RTTs[i]
		count++
	}

	if count == 0 {
		return -1 * time.Millisecond
	}

	return sum / time.Duration(count)
}

type Server struct {
	sync.RWMutex

	Network *Network

	Addr string
	ID   string

	connectedClients map[string]*ConnectedClientInfo
}

// NewServer creates the server with a given address.
func NewServer(addr string) *Server {
	serverID := uuid.NewString()
	s := &Server{
		Addr:             addr,
		Network:          NewNetwork(serverID),
		ID:               serverID,
		connectedClients: make(map[string]*ConnectedClientInfo),
	}

	go s.startHeartbeats()
	return s
}

func (s *Server) startHeartbeats() {
	for {
		s.Lock()

		for clientID, info := range s.connectedClients {
			client, ok := s.Network.GetClient(clientID)

			if !ok || time.Since(info.LastHeartbeat) >= timeoutInterval {
				arcade.ViewManager.ProcessEvent(NewClientDisconnectEvent(clientID))
				delete(s.connectedClients, clientID)
				continue
			}

			metadata := arcade.ViewManager.GetHeartbeatMetadata()

			client.Lock()
			s.Network.Send(client, NewHeartbeatMessage(client.Seq, metadata))
			s.connectedClients[clientID].HeartbeatSendTimes[client.Seq] = time.Now()
			client.Seq++
			client.Unlock()
		}

		s.Unlock()

		<-time.After(heartbeatInterval)
	}
}

func (s *Server) BeginHeartbeats(clientID string) {
	s.Lock()
	defer s.Unlock()

	s.connectedClients[clientID] = &ConnectedClientInfo{
		LastHeartbeat:      time.Now(),
		HeartbeatSendTimes: make(map[int]time.Time),
		RTTs:               make([]time.Duration, 0),
	}
}

func (s *Server) GetHeartbeatClients() map[string]*ConnectedClientInfo {
	s.RLock()
	defer s.RUnlock()

	return s.connectedClients
}

// connect connects to a client at a given address.
func (s *Server) connect(c *Client) error {
	sess, err := kcp.Dial(c.Addr)

	if err != nil {
		return err
	}

	c.start(sess)

	c.connectedCh = make(chan bool)
	s.Network.Send(c, NewPingMessage())

	// Timeout if we don't receive a response
	time.AfterFunc(timeoutInterval, func() {
		c.connectedMux.Lock()
		defer c.connectedMux.Unlock()

		if !c.connected {
			c.timedOut = true
			c.connectedCh <- false
		}
	})

	if !<-c.connectedCh {
		return errors.New("client timed out")
	}

	go s.Network.PropagateRoutes()

	return nil
}

func (s *Server) handleMessage(c *Client, data []byte) {
	p, err := parseMessage(data)

	if err != nil {
		// Most likely malformed message/packet is too large, ignore
		return
	}

	msg := reflect.ValueOf(p).FieldByName("Message").Interface().(Message)

	if arcade.Distributor {
		fmt.Println(msg)
		fmt.Printf("Received '%s' from %s\n", msg.Type, msg.SenderID[:4])

		if msg.Type == "error" {
			fmt.Println(p)
		}
	}

	// Process message and prepare response
	var res interface{}

	switch p := p.(type) {
	case DisconnectMessage:
		arcade.ViewManager.ProcessEvent(&ClientDisconnectEvent{
			ClientID: c.ID,
		})

		s.Network.DeleteClient(c.ID)
	case PingMessage:
		c.ID = msg.SenderID
		c.ClientRoutingInfo = ClientRoutingInfo{
			Distance: 1,
		}
		c.Neighbor = true

		s.Network.AddClient(c)

		res = NewPongMessage(arcade.Distributor)
	case PongMessage:
		c.ID = msg.SenderID
		c.ClientRoutingInfo = ClientRoutingInfo{
			Distance:    1,
			Distributor: p.Distributor,
		}
		c.Neighbor = true

		s.Network.AddClient(c)

		c.connectedMux.Lock()
		if !c.timedOut {
			c.connected = true
			c.connectedCh <- true
		}
		c.connectedMux.Unlock()
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

			switch p := p.(type) {
			case HeartbeatMessage:
				s.Lock()
				if _, ok := s.connectedClients[msg.SenderID]; ok {
					s.connectedClients[msg.SenderID].LastHeartbeat = time.Now()
				}
				s.Unlock()

				// Send heartbeat metadata to view
				arcade.ViewManager.ProcessEvent(NewHeartbeatEvent(p.Metadata))

				// Reply to heartbeat
				res = NewHeartbeatReplyMessage(p.Seq)
			case HeartbeatReplyMessage:
				if msg.RecipientID == s.ID {
					s.Lock()
					if _, ok := s.connectedClients[msg.SenderID]; ok {
						s.connectedClients[msg.SenderID].LastHeartbeat = time.Now()
						s.connectedClients[msg.SenderID].RTTs = append(s.connectedClients[msg.SenderID].RTTs, time.Since(s.connectedClients[msg.SenderID].HeartbeatSendTimes[p.Seq]))
					}
					s.Unlock()

					arcade.ViewManager.RequestDebugRender()
				}
			default:
				res = processMessage(c, p)
			}
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
	reflect.ValueOf(res).Elem().FieldByName("Message").FieldByName("MessageID").Set(reflect.ValueOf(msg.MessageID))

	resData, err := res.(encoding.BinaryMarshaler).MarshalBinary()

	if err != nil {
		panic(err)
	}

	c.sendCh <- resData
}

func (s *Server) startWithNextOpenPort() {
	for {
		s.Addr = fmt.Sprintf("0.0.0.0:%d", arcade.Port)
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

func (s *Server) ScanLAN() {
	ips, err := GetLANIPs()

	if err != nil {
		panic(err)
	}

	for _, ip := range ips {
		client := NewNeighboringClient(fmt.Sprintf("%s:6824", ip))
		go arcade.Server.connect(client)

		time.Sleep(time.Millisecond)
	}
}
