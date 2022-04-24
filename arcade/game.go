package arcade

import (
	"fmt"
	"sync"
)

const (
	Pong = "Pong"
	Tron = "Tron"
)

type Game struct {
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

func CreateGame(name string, private bool, gameType string, capacity int ) *Game {
	return &Game{Name: name, Private: private, GameType: gameType, Capacity: capacity, NumFull: 1}
}

func GameStart() {
	fmt.Println("hello world")
}

func GenerateCode() {
	fmt.Println("hello world")
}

func (g *Game) AddPlayer(newPlayer Player) {
	g.mu.Lock()
	g.PlayerList = append(g.PlayerList, newPlayer)
	g.mu.Unlock()
}
