package arcade

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/xtaci/kcp-go/v5"
)

type Server struct {
	sync.RWMutex

	Addr string
	ID   string

	clients map[string]*Client
}

// NewServer creates the server with a given address.
func NewServer(addr string) *Server {
	return &Server{
		Addr:    addr,
		ID:      uuid.NewString(),
		clients: make(map[string]*Client, 0),
	}
}

func (s *Server) connectToNextOpenPort() {
	for port := 6824; port < 6824+10; port++ {
		if port == hostPort {
			port++
			continue
		}

		client := NewClient(fmt.Sprintf("127.0.0.1:%d", port))

		if err := server.connect(client); err != nil {
			continue
		}

		client.send(NewHelloMessage())
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
	c.send(NewPingMessage(server.ID))

	// TODO: timeout if no response
	if !<-c.connectedCh {
		return errors.New("client failed to connect")
	}

	return nil
}

func (s *Server) startWithNextOpenPort() {
	hostPort = 6824

	for {
		s.Addr = fmt.Sprintf("127.0.0.1:%d", hostPort)
		s.start()

		hostPort++
	}
}

// startServer starts listening for connections on a given address.
func (s *Server) start() error {
	listener, err := kcp.Listen(s.Addr)

	if err != nil {
		return err
	}

	fmt.Printf("Listening at %s...\n", s.Addr)

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

func (s *Server) getClients() map[string]float64 {
	s.RLock()
	defer s.RUnlock()

	clients := server.clients
	clientPings := make(map[string]float64, len(clients))

	for i := range clients {
		if !clients[i].Neighbor || clients[i].Distributor {
			continue
		}

		clientPings[clients[i].ID] = 1
	}

	return clientPings
}

func (s *Server) AddClients(distributor *Client, clients map[string]float64) {
	s.Lock()
	defer s.Unlock()

	for clientID := range clients {
		s.clients[clientID] = &Client{
			Addr:            distributor.Addr,
			ID:              clientID,
			sendCh:          distributor.sendCh,
			pendingMessages: make(map[string]chan interface{}),
		}
	}
}

func (s *Server) SendToAllClients(msg interface{}) {
	s.RLock()
	defer s.Unlock()

	for _, client := range s.clients {
		if client.Distributor {
			continue
		}
		client.send(msg)
	}
}

