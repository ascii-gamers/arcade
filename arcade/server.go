package arcade

import (
	"fmt"
	"log"
	"sync"

	"github.com/xtaci/kcp-go/v5"
)

const maxBufferSize = 1024

type Server struct {
	sync.RWMutex

	Addr    string
	clients []*Client
}

// NewServer creates the server with a given address.
func NewServer(addr string) *Server {
	return &Server{
		Addr:    addr,
		clients: make([]*Client, 0),
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

	s.Lock()
	s.clients = append(s.clients, c)
	s.Unlock()

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
