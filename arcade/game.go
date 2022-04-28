package arcade

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
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

type PendingGame struct {
	Name string
	Code string
	Private bool
	GameType string 
	Capacity int
	NumFull      int
	PlayerList   []*Player
	mu           sync.Mutex
	Host string
}

type Game[GS any, CS any] struct {
	Name         string
	PlayerList   []*Player
	mu           sync.Mutex

	Me string
	GameState GS
	// clientIps []string
	ClientStates map[string]CS
	Started bool
	Host string
	HostSyncPeriod int 
	TimestepPeriod int
	Timestep int
}

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func CreatePendingGame(name string, private bool, gameType string, capacity int ) *PendingGame {
	game := PendingGame{Name: name, Private: private, GameType: gameType, Capacity: capacity, NumFull: 1}
	if private {
		game.GenerateCode()
	}
	return &game
}

func GameStart() {
	fmt.Println("hello world")
}

func (g *PendingGame) GenerateCode() string{
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

func (g *PendingGame) AddPlayer(newPlayer Player) {
	g.mu.Lock()
	g.PlayerList = append(g.PlayerList, &newPlayer)
	g.mu.Unlock()
}


func CreateGame(pendingGame *PendingGame) {
	switch pendingGame.GameType {
	case Tron:
		mgr.SetView(NewTronGame(pendingGame))
	}
}

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

func (g *Game[GS, CS]) start() {
	g.Started = true
	if g.Me == g.Host && g.HostSyncPeriod > 0 {
		go g.startHostSync()
	}
}

func (g *Game[GS, CS]) startHostSync() {
	for g.Started {
		time.Sleep(time.Duration(g.HostSyncPeriod))
		g.sendGameUpdate()
	}
}

func (g *Game[GS, CS]) sendClientUpdate(update CS) {
	g.ClientStates[g.Me] = update
	for clientIp := range g.ClientStates {
		n.send(clientIp, update)
	}
}

func (g *Game[GS, CS]) sendGameUpdate() {
	if g.Me == g.Host {
		for clientIp := range g.ClientStates {
			if clientIp != g.Me {
				data := GameUpdateData[GS, CS]{g.GameState, g.ClientStates}
				n.send(clientIp, data)
			}
		}
	}
}

func (g *Game[GS, CS]) handleClientUpdate(clientIp string, data ClientUpdateData[CS]) {
	g.ClientStates[clientIp] = data.update
}

func (g *Game[GS, CS]) handleGameUpdate(clientIp string, data GameUpdateData[GS, CS]) {
	g.GameState = data.gameUpdate
	g.ClientStates = data.clientStates
}
