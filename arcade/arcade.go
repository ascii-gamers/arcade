package arcade

import (
	"flag"
	"fmt"
	"os"
	"time"
)

var distributor = false
var mgr = NewViewManager()
var server *Server
var hostPort int
var lobby *Lobby

func Start() {
	dist := flag.Bool("distributor", false, "Run as a distributor")
	flag.BoolVar(dist, "d", false, "Run as a distributor")

	distributorAddr := flag.String("distributor-addr", "54.80.111.42:6824", "Distributor address")
	flag.StringVar(distributorAddr, "da", "54.80.111.42:6824", "Distributor address")

	port := flag.Int("port", 6824, "Port to listen on")
	flag.IntVar(port, "p", 6824, "Port to listen on")

	flag.Parse()

	distributor = *dist
	hostPort = *port

	// Start host server
	server = NewServer("")

	if distributor {
		server.Addr = fmt.Sprintf("0.0.0.0:%d", hostPort)
		server.start()
		os.Exit(0)
	}

	go server.startWithNextOpenPort()

	// TODO: Make better solution for this later -- wait for server to start
	time.Sleep(10 * time.Millisecond)

	go server.connect(NewClient(*distributorAddr))

	// TODO: Make better solution for this later -- wait to connect to distributor
	time.Sleep(10 * time.Millisecond)

	// Start view manager
	splashView := NewSplashView()
	mgr.Start(splashView)
}
