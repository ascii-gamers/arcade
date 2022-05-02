package arcade

import (
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

type Game[GS any, CS any] struct {
	Name      string
	PlayerIDs []string
	mu        sync.Mutex

	Me        string
	GameState GS
	// clientIps []string
	ClientStates   map[string]CS
	Started        bool
	HostID         string
	HostSyncPeriod int
	TimestepPeriod int
	Timestep       int
}

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func NewGame(lobby *Lobby) {
	switch lobby.GameType {
	case Tron:
		mgr.SetView(NewTronGameView(lobby))
	}
}

type ClientUpdateData[CS any] struct {
	Id     string
	Update CS
}

type GameUpdateData[GS any, CS any] struct {
	GameUpdate   GS
	ClientStates map[string]CS
}

func (g *Game[GS, CS]) start() {
	g.Started = true
	if g.Me == g.HostID && g.HostSyncPeriod > 0 {
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
	clientUpdate := ClientUpdateData[CS]{g.Me, update}

	for clientId := range g.ClientStates {
		if client, ok := server.Network.GetClient(clientId); ok && clientId != g.Me {
			server.Network.Send(client, clientUpdate)
		}
	}
}

func (g *Game[GS, CS]) sendGameUpdate() {
	if g.Me != g.HostID {
		return
	}

	server.RLock()
	defer server.RUnlock()

	for clientID := range g.ClientStates {
		if client, ok := server.Network.GetClient(clientID); ok {
			data := GameUpdateData[GS, CS]{g.GameState, g.ClientStates}
			server.Network.Send(client, data)
		}
	}
}

func (g *Game[GS, CS]) handleClientUpdate(clientIp string, data ClientUpdateData[CS]) {
	g.ClientStates[clientIp] = data.Update
}

func (g *Game[GS, CS]) handleGameUpdate(clientIp string, data GameUpdateData[GS, CS]) {
	g.GameState = data.GameUpdate
	g.ClientStates = data.ClientStates
}
