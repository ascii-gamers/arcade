package arcade

import (
	"os"
	"time"
)

var distributor = false
var mgr = NewViewManager()
var server *Server
var hostPort int
var lobby *Lobby

func Start() {
	if len(os.Args) > 1 && (os.Args[1] == "-d" || os.Args[1] == "--distributor") {
		distributor = true
	}

	// Start host server
	server = NewServer("")

	if distributor {
		server.Addr = "127.0.0.1:8000"
		server.start()
		os.Exit(0)
	}

	go server.startWithNextOpenPort()

	// TODO: Make better solution for this later -- wait for server to start
	time.Sleep(10 * time.Millisecond)

	go server.connect(NewClient("127.0.0.1:8000"))

	// TODO: Make better solution for this later -- wait to connect to distributor
	time.Sleep(10 * time.Millisecond)

	// Start view manager
	splashView := NewSplashView()
	mgr.Start(splashView)
}
