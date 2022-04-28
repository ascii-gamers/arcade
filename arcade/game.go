package arcade

import (
	"fmt"
	"math/rand"
	"sync"
)

const (
	Pong = "Pong"
	Tron = "Tron"
)

var pong_graphic_double_1 = []string{
	"o      .   _______ _______		  ",
	"\\_ 0     /______//______/|   @_o",
	"  /\\_,  /______//______/     /\\",
	" | \\    |      ||      |     / |",
}

var pong_graphic_double_2 = []string{
	"  o        _______ _______		  ",
	" /_ 0     /__ .__//______/|   @_o",
	"  /\\_,  /______//______/     /\\",
	" | \\    |      ||      |     / |",
}

var pong_graphic_single_1 = []string{
	"o      .   _______ _______		  ",
	"\\_ 0     /______//______/|      ",
	"  /\\_,  /______//______/        ",
	" | \\    |      ||      |        ",
}

var pong_graphic_single_2 = []string{
	"  o        _______ _______		  ",
	" /_ 0     /__ .__//______/|      ",
	"  /\\_,  /______//______/        ",
	" | \\    |      ||      |        ",
}

var tron_graphic = []string{
	"						 ____		 ",
	"________________________/ O  \\___/",
    "<_____________________________/   \\",
}

var pong_header = []string{	
	"█▀█ █▀█ █▄░█ █▀▀",
	"█▀▀ █▄█ █░▀█ █▄█",
}

var tron_header = []string{
	"▀█▀ █▀█ █▀█ █▄░█",
	"░█░ █▀▄ █▄█ █░▀█",
}


type Game struct {
	Name         string
	Code string
	Private      bool
	GameType     string
	Capacity     int
	NumFull      int
	TerminalSize int
	PlayerList   []*Player
	mu           sync.Mutex
}

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func CreateGame(name string, private bool, gameType string, capacity int ) *Game {
	game := Game{Name: name, Private: private, GameType: gameType, Capacity: capacity, NumFull: 1}
	if private {
		game.GenerateCode()
	}
	return &game
}

func GameStart() {
	fmt.Println("hello world")
}

func (g *Game) GenerateCode() string{
	// see if code already exists
	g.mu.Lock()
	code := g.Code
	g.mu.Unlock()
	if len(code) > 0 {
		return code
	}
	for i:= 0; i < 4; i ++ {
		v := rand.Intn(25)
		code += string(letters[v])
	}
	g.mu.Lock()
	g.Code = code
	g.mu.Unlock()
	return code
}

func (g *Game) AddPlayer(newPlayer *Player) {
	g.mu.Lock()
	g.PlayerList = append(g.PlayerList, newPlayer)
	g.mu.Unlock()
}
