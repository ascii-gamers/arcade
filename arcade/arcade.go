package arcade

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func Start() {
	port, _ := strconv.ParseInt(os.Args[1], 10, 64)
	hostPort, _ := strconv.ParseInt(os.Args[2], 10, 64)

	server := NewServer(fmt.Sprintf("127.0.0.1:%d", port))
	go server.start()

	client := NewClient(fmt.Sprintf("127.0.0.1:%d", hostPort))
	go server.connect(client)

	if port == 8080 {
		for {
			client.send(NewHelloMessage())
			time.Sleep(time.Second)
		}
	}

	<-make(chan bool, 1)
}
