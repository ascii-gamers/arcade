package arcade

import "time"

var mgr = NewViewManager()
var server *Server
var hostPort int

func Start() {
	// Start host server
	server = NewServer("")
	go server.startWithNextOpenPort()

	// TODO: Make better solution for this later -- wait for server to start
	time.Sleep(10 * time.Millisecond)

	// Start view manager
	gamesListView := NewGamesListView()
	mgr.Start(gamesListView)
}
