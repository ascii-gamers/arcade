package arcade

import (
	"flag"
	"fmt"
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

		Port: 6824,
	}
}

func Start() {
	dist := flag.Bool("distributor", false, "Run as a distributor")
	flag.BoolVar(dist, "d", false, "Run as a distributor")

	distributorAddr := flag.String("distributor-addr", "54.80.111.42:6824", "Distributor address")
	flag.StringVar(distributorAddr, "da", "54.80.111.42:6824", "Distributor address")

	port := flag.Int("port", 6824, "Port to listen on")
	flag.IntVar(port, "p", 6824, "Port to listen on")

	lan := flag.Bool("lan", false, "Scan local network for clients")

	test := flag.Bool("t", false, "Test mode")
	flag.Parse()

	arcade.Distributor = *dist
	arcade.LAN = *lan
	arcade.Port = *port

	// Start host server
	arcade.Server = NewServer("")
	arcade.Lobby = &Lobby{}

	// go arcade.Server.ScanLAN()
	// time.Sleep(10 * time.Second)

	if arcade.Distributor {
		arcade.Server.Addr = fmt.Sprintf("0.0.0.0:%d", arcade.Port)
		arcade.Server.start()
		os.Exit(0)
	}

	go arcade.Server.startWithNextOpenPort()

	// TODO: Make better solution for this later -- wait for server to start
	time.Sleep(10 * time.Millisecond)

	client := NewNeighboringClient(*distributorAddr)
	go arcade.Server.Connect(client)

	// TODO: Make better solution for this later -- wait to connect to distributor
	time.Sleep(10 * time.Millisecond)

	if *test {
		fmt.Println("sending...")
		res, err := arcade.Server.Network.SendAndReceive(client, NewPingMessage())
		fmt.Println(res, err)
		time.Sleep(10 * time.Second)
	}

	// Start view manager
	splashView := NewSplashView()
	arcade.ViewManager.Start(splashView)
}
