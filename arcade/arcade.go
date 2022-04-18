package arcade

import (
	"fmt"
)

var mgr = NewViewManager()

func Start() {
	// port, _ := strconv.ParseInt(os.Args[1], 10, 64)
	// hostPort, _ := strconv.ParseInt(os.Args[2], 10, 64)

	server := NewServer(fmt.Sprintf("127.0.0.1:%d", 9000))
	go server.start()

	client := NewClient(fmt.Sprintf("127.0.0.1:%d", 9001))
	go server.connect(client)

	gamesListView := NewGamesListView()
	mgr.Start(gamesListView)
}
