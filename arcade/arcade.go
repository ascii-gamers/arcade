package arcade

import (
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
	flag.Parse()

	arcade.Distributor = *dist
	arcade.Port = *port

	// Start host server
	arcade.Server = NewServer("")
	arcade.Lobby = &Lobby{}

	if arcade.Distributor {
		arcade.Server.Addr = fmt.Sprintf("0.0.0.0:%d", arcade.Port)
		arcade.Server.start()
		os.Exit(0)
	}

	go arcade.Server.startWithNextOpenPort()

	// TODO: Make better solution for this later -- wait for server to start
	time.Sleep(10 * time.Millisecond)

	// Create log file
	f, err := os.OpenFile(fmt.Sprintf("log-%d", arcade.Port), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		panic(err)
	}

	defer f.Close()
	log.SetOutput(f)

	// Connect to distributor
	client := NewNeighboringClient(*distributorAddr)
	go arcade.Server.Connect(client)

	// TODO: Make better solution for this later -- wait to connect to distributor
	time.Sleep(10 * time.Millisecond)

	// Start view manager
	splashView := NewSplashView()
	arcade.ViewManager.Start(splashView)
}
