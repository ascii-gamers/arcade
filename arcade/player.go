package arcade

import "fmt"

const (
	GameDone = "GameDone"
	Waiting  = "Waiting"
	Looking  = "Looking"
	Playing  = "Playing"
)

type Player struct {
	Username string
	Status   string
	IP       string
}

func PlayerStart() {
	fmt.Println("hello world")
}
