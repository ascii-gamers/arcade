package arcade

import (
	"fmt"
	"sync"
)

const (
	Pong = "Pong"
	Tron = "Tron"
)

type GameInfo struct {
	Name         string
	Private      bool
	GameType     string
	Capacity     int
	NumFull      int
	TerminalSize int
	PlayerList   []Player
	mu           sync.Mutex
}

type Lobby struct {
	mu sync.Mutex
}

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func GameStart() {
	fmt.Println("hello world")
}

func GenerateCode() {
	fmt.Println("hello world")
}

func (g *GameInfo) AddPlayer(newPlayer Player) {
	g.mu.Lock()
	g.PlayerList = append(g.PlayerList, newPlayer)
	g.mu.Unlock()
}
