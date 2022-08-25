package arcade

import (
	"arcade/arcade/message"
	"arcade/arcade/multicast"
	"arcade/arcade/net"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xtaci/kcp-go/v5"
)

const timeoutInterval = 2500 * time.Millisecond
const heartbeatInterval = 250 * time.Millisecond
const rttAverageNum = 10

type ConnectedClientInfo struct {
	LastHeartbeat time.Time
	RTTs          []time.Duration
}

func (c *ConnectedClientInfo) GetMeanRTT() time.Duration {
	var sum time.Duration
	count := 0

	for i := len(c.RTTs) - 1; i >= 0 && i >= len(c.RTTs)-(rttAverageNum+1); i-- {
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
	mgr *ViewManager

	Network *net.Network

	Addr string
	ID   string

	connectedClients map[string]*ConnectedClientInfo

	// Message IDs to ping times
	pingMessageTimes map[string]time.Time
}

// NewServer creates the server with a given address.
func NewServer(addr string, port int, distributor bool, mgr *ViewManager) *Server {
	id := uuid.NewString()
	net := net.NewNetwork(id, port, distributor)

	s := &Server{
		mgr:              mgr,
		Addr:             addr,
		Network:          net,
		ID:               id,
		connectedClients: make(map[string]*ConnectedClientInfo),
		pingMessageTimes: make(map[string]time.Time),
	}

	message.AddListener(message.Listener{
		Distributor: true,
		ServerID:    id,
		Handle:      s.handleMessage,
	})

	go s.startHeartbeats()

	return s
}

func (s *Server) startHeartbeats() {
	for {
		for clientID, info := range s.connectedClients {
			client, ok := s.Network.GetClient(clientID)

			if !ok || time.Since(info.LastHeartbeat) >= timeoutInterval {
				s.Network.Disconnect(clientID)

				s.Lock()
				delete(s.connectedClients, clientID)
				s.Unlock()
				continue
			}

			metadata := s.mgr.GetHeartbeatMetadata()

			go func(clientID string) {
				start := time.Now()
				res, err := s.Network.SendAndReceive(client, NewHeartbeatMessage(0, metadata))
				end := time.Now()

				_, ok := res.(*HeartbeatReplyMessage)

				if !ok || err != nil {
					return
				}

				s.Lock()
				if _, ok := s.connectedClients[clientID]; ok {
					s.connectedClients[clientID].RTTs = append(s.connectedClients[clientID].RTTs, end.Sub(start))
					s.connectedClients[clientID].LastHeartbeat = time.Now()
				}
				s.Unlock()
			}(clientID)
		}

		<-time.After(heartbeatInterval)
	}
}

func (s *Server) BeginHeartbeats(clientID string) {
	s.Lock()
	defer s.Unlock()

	s.connectedClients[clientID] = &ConnectedClientInfo{
		LastHeartbeat: time.Now(),
		RTTs:          make([]time.Duration, 0),
	}
}

func (s *Server) EndHeartbeats(clientID string) {
	s.Lock()
	defer s.Unlock()

	delete(s.connectedClients, clientID)
}

func (s *Server) EndAllHeartbeats() {
	s.Lock()
	defer s.Unlock()

	s.connectedClients = make(map[string]*ConnectedClientInfo)
}

func (s *Server) GetHeartbeatClients() map[string]*ConnectedClientInfo {
	s.RLock()
	defer s.RUnlock()

	return s.connectedClients
}

func (s *Server) handleMessage(client, msg interface{}) interface{} {
	c := client.(*net.Client)

	baseMsg := reflect.ValueOf(msg).Elem().FieldByName("Message").Interface().(message.Message)

	// Ping messages may not have a recipient ID set
	if baseMsg.RecipientID == "" {
		baseMsg.RecipientID = s.ID
	}

	// Signal message received if necessary
	s.Network.SignalReceived(baseMsg.MessageID, msg)

	if arcade.Distributor {
		fmt.Println(msg)
		fmt.Printf("Received '%s' from %s\n", baseMsg.Type, baseMsg.SenderID[:4])

		if baseMsg.Type == "error" {
			fmt.Println(msg)
		}
	}

	// Process message and return response
	switch msg := msg.(type) {
	case *DisconnectMessage:
		s.Network.Disconnect(c.ID)
	case *net.PingMessage, *net.PongMessage, *net.RoutingMessage:
		break
	default:
		if baseMsg.RecipientID != s.ID {
			if arcade.Distributor {
				fmt.Println("Forwarding message to", baseMsg.RecipientID[:4])
				fmt.Println(msg)
			}

			s.RLock()
			recipient, ok := s.Network.GetClient(baseMsg.RecipientID)
			s.RUnlock()

			if ok {
				recipient.Send(msg)
				// s.Network.Send(recipient, msg)
				return nil
			} else {
				return NewErrorMessage("invalid recipient")
			}
		} else {
			if arcade.Distributor {
				fmt.Println(msg)
				panic("Recipient: " + baseMsg.RecipientID + ", self: " + s.ID)
			}

			switch msg := msg.(type) {
			case *HeartbeatMessage:
				s.Lock()
				if _, ok := s.connectedClients[msg.SenderID]; ok {
					s.connectedClients[msg.SenderID].LastHeartbeat = time.Now()
					c.Distance = float64(s.connectedClients[msg.SenderID].GetMeanRTT().Milliseconds())
				}
				s.Unlock()

				// Send heartbeat metadata to view
				s.mgr.ProcessEvent(NewHeartbeatEvent(msg.Metadata))

				// Reply to heartbeat
				return NewHeartbeatReplyMessage(msg.Seq)
			default:
				return s.mgr.ProcessMessage(c, msg)
			}
		}
	}

	return nil
}

// Start starts listening for connections on a given address.
func (s *Server) Start() error {
	listener, err := kcp.Listen(s.Addr)

	if err != nil {
		panic(err)
	}

	fmt.Printf("Listening at %s...\n", s.Addr)
	fmt.Printf("ID: %s\n", s.ID)

	go multicast.Listen(s.ID, s)

	for {
		// Wait for new client connections
		conn, err := listener.Accept()

		if err != nil {
			panic(err)
		}

		s.Network.Connect(conn.RemoteAddr().String(), "", conn)
	}
}

//
// MulticastDelegate methods
//

func (s *Server) ClientDiscovered(addr, id string) {
	s.RLock()
	defer s.RUnlock()

	s.Network.Connect(addr, id, nil)
}
