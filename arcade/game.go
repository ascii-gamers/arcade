package arcade

import (
	"arcade/arcade/message"
	"arcade/raft"
	"encoding/json"
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

type Game[GS any, CS any] struct {
	ID        string
	Name      string
	PlayerIDs []string
	// mu        sync.Mutex

	Me string
	// GameState GS
	// clientIps []string
	// ClientStates   map[string]CS
	Started        bool
	HostID         string
	HostSyncPeriod int
	TimestepPeriod int
	Timestep       int
	RaftServer     *raft.Raft
}

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func NewGame(mgr *ViewManager, lobby *Lobby) {
	switch lobby.GameType {
	case Tron:
		mgr.SetView(NewTronGameView(mgr, lobby))
	}
}

type ClientUpdateMessage[CS any] struct {
	message.Message
	Id     string
	Update CS
}

type GameUpdateMessage[GS any, CS any] struct {
	message.Message
	GameUpdate GS
	// ClientStates map[string]CS
	LastInps    map[string]int
	ID          string
	FragmentNum int
}

type AckGameUpdateMessage struct {
	message.Message
}

type StartGameMessage struct {
	message.Message
	GameID string
}

type EndGameMessage struct {
	message.Message
	Winner string
}

func NewEndGameMessage(winner string) *EndGameMessage {
	return &EndGameMessage{message.Message{Type: "end_game"}, winner}
}

func NewStartGameMessage(GameID string) *StartGameMessage {
	return &StartGameMessage{message.Message{Type: "start_game"}, GameID}
}

func NewAckGameUpdateMessage() *AckGameUpdateMessage {
	return &AckGameUpdateMessage{message.Message{Type: "ack_game_update"}}
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
