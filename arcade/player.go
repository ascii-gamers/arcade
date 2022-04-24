package arcade

import "fmt"

const (
	GameDone = "GameDone"
	Waiting  = "Waiting"
	Looking  = "Looking"
	Playing  = "Playing"
)

type Player struct {
	Client
	Username string
	Status   string
	Host bool
}

func PlayerStart() {
	fmt.Println("hello world")
}
