package arcade

import (
	"encoding/json"
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

type Game[GS any, CS any] struct {
	ID        string
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
		arcade.ViewManager.SetView(NewTronGameView(lobby))
	}
}

type ClientUpdateMessage[CS any] struct {
	Message
	Id     string
	Update CS
}

type GameUpdateMessage[GS any, CS any] struct {
	Message
	GameUpdate GS
	// ClientStates map[string]CS
	LastInps    map[string]int
	ID          string
	FragmentNum int
}

type AckGameUpdateMessage struct {
	Message
}

type StartGameMessage struct {
	Message
	GameID string
}

type EndGameMessage struct {
	Message
	Winner string
}

func NewEndGameMessage(winner string) *EndGameMessage {
	return &EndGameMessage{Message{Type: "end_game"}, winner}
}

func NewStartGameMessage(GameID string) *StartGameMessage {
	return &StartGameMessage{Message{Type: "start_game"}, GameID}
}

func NewAckGameUpdateMessage() *AckGameUpdateMessage {
	return &AckGameUpdateMessage{Message{Type: "ack_game_update"}}
}

func (m ClientUpdateMessage[any]) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m GameUpdateMessage[GS, CS]) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m StartGameMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m AckGameUpdateMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m EndGameMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (g *Game[GS, CS]) start() {
	g.Started = true
	// if g.Me == g.HostID && g.HostSyncPeriod > 0 {
	// 	go g.startHostSync()
	// }
}

// func (g *Game[GS, CS]) startHostSync() {
// 	for g.Started {
// 		time.Sleep(time.Duration(g.HostSyncPeriod * int(time.Millisecond)))
// 		g.sendGameUpdate()
// 	}
// }

// use these to generalize funcs in tron game

// func (g *Game[GS, CS]) handleClientUpdate(clientIp string, data ClientUpdateMessage[CS]) {
// 	g.ClientStates[clientIp] = data.Update
// }

// func (g *Game[GS, CS]) handleGameUpdate(clientIp string, data GameUpdateMessage[GS, CS]) {
// 	g.GameState = data.GameUpdate
// 	g.ClientStates = data.ClientStates
// }
