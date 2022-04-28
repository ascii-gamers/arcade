package arcade

import (
<<<<<<< Updated upstream
	"fmt"
	"sync"
)

const (
	Pong = "Pong"
	Tron = "Tron"
)

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
	return &Game{Name: name, Private: private, GameType: gameType, Capacity: capacity, NumFull: 1}
}

func GameStart() {
	fmt.Println("hello world")
}

func GenerateCode() {
	fmt.Println("hello world")
}

func (g *Game) AddPlayer(newPlayer *Player) {
	g.mu.Lock()
	g.PlayerList = append(g.PlayerList, newPlayer)
	g.mu.Unlock()
}
=======
	"time"
)

type Networking struct {

}

var n = Networking{}

func (n Networking) send(ip string, data any) {
	return;
}

type ClientUpdateData[CS any] struct {
	update CS
}

type GameUpdateData[GS any, CS any] struct {
	gameUpdate GS
	clientStates map[string]CS
}

type Game[GS any, CS any] struct {
	me string
	gameState GS
	clientIps []string
	clientStates map[string]CS
	started bool
	capacity int
	host string
	hostSyncPeriod int 
	timestepPeriod int
	timestep int
}

func (g *Game[GS, CS]) start() {
	g.started = true
	if g.me == g.host && g.hostSyncPeriod > 0 {
		go g.startHostSync()
	}
}

func (g *Game[GS, CS]) startHostSync() {
	for g.started {
		time.Sleep(time.Duration(g.hostSyncPeriod))
		g.sendGameUpdate()
	}
}

func (g *Game[GS, CS]) sendClientUpdate(update CS) {
	g.clientStates[g.me] = update
	for _, clientIp := range g.clientIps {
		n.send(clientIp, update)
	}
}

func (g *Game[GS, CS]) sendGameUpdate() {
	if g.me == g.host {
		for _, clientIp := range g.clientIps {
			if clientIp != g.me {
				data := GameUpdateData[GS, CS]{g.gameState, g.clientStates}
				n.send(clientIp, data)
			}
		}
	}
}

func (g *Game[GS, CS]) handleClientUpdate(clientIp string, data ClientUpdateData[CS]) {
	g.clientStates[clientIp] = data.update
}

func (g *Game[GS, CS]) handleGameUpdate(clientIp string, data GameUpdateData[GS, CS]) {
	g.gameState = data.gameUpdate
	g.clientStates = data.clientStates
}
>>>>>>> Stashed changes
