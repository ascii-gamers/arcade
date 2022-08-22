package arcade

import (
	"arcade/arcade/message"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type Arcade struct {
	Distributor bool
	Port        int
	LAN         bool

	ViewManager *ViewManager

	Server *Server

	lobbyMux sync.RWMutex
	Lobby    *Lobby
}

var arcade = NewArcade()

func NewArcade() *Arcade {
	return &Arcade{
		Distributor: false,
		ViewManager: NewViewManager(),
	}
}

func Start() {
	dist := flag.Bool("distributor", false, "Run as a distributor")
	flag.BoolVar(dist, "d", false, "Run as a distributor")

	distributorAddr := flag.String("distributor-addr", "54.80.111.42:6824", "Distributor address")
	flag.StringVar(distributorAddr, "da", "54.80.111.42:6824", "Distributor address")

	port := flag.Int("port", 6824, "Port to listen on")
	flag.IntVar(port, "p", 6824, "Port to listen on")
	flag.Parse()

	// Create log file
	f, err := os.OpenFile(fmt.Sprintf("log-%d", *port), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		panic(err)
	}

	defer f.Close()
	log.SetOutput(f)

	// Register messages
	message.Register(DisconnectMessage{Message: message.Message{Type: "disconnect"}})
	message.Register(HeartbeatMessage{Message: message.Message{Type: "heartbeat"}})
	message.Register(HeartbeatReplyMessage{Message: message.Message{Type: "heartbeat_reply"}})
	message.Register(HelloMessage{Message: message.Message{Type: "hello"}})
	message.Register(JoinMessage{Message: message.Message{Type: "join"}})
	message.Register(JoinReplyMessage{Message: message.Message{Type: "join_reply"}})
	message.Register(LobbyEndMessage{Message: message.Message{Type: "lobby_end"}})
	message.Register(LobbyInfoMessage{Message: message.Message{Type: "lobby_info"}})

	arcade.Distributor = *dist
	arcade.Port = *port

	// Start host server
	arcade.Server = NewServer(fmt.Sprintf("0.0.0.0:%d", *port), *port)
	arcade.Lobby = &Lobby{}

	if arcade.Distributor {
		arcade.Server.Addr = fmt.Sprintf("0.0.0.0:%d", arcade.Port)
		arcade.Server.start()
		os.Exit(0)
	}

	go arcade.Server.start()

	// TODO: Make better solution for this later -- wait for server to start
	time.Sleep(10 * time.Millisecond)

	// Connect to distributor
	go arcade.Server.Network.Connect(*distributorAddr, nil)

	// TODO: Make better solution for this later -- wait to connect to distributor
	time.Sleep(10 * time.Millisecond)

	// Start view manager
	splashView := NewSplashView()
	arcade.ViewManager.Start(splashView)
}
