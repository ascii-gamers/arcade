package arcade

import (
	"encoding"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"
	"unicode/utf8"

	"arcade/arcade/message"
	"arcade/arcade/net"
	"arcade/raft"

	"github.com/gdamore/tcell/v2"
	"github.com/google/uuid"
	"github.com/jinzhu/copier"
)

var COUNTDOWN_1 = []string{
	"░░███╗░░",
	"░████║░░",
	"██╔██║░░",
	"╚═╝██║░░",
	"███████╗",
	"╚══════╝",
}

var COUNTDOWN_2 = []string{
	"██████╗░",
	"╚════██╗",
	"░░███╔═╝",
	"██╔══╝░░",
	"███████╗",
	"╚══════╝",
}

var COUNTDOWN_3 = []string{
	"██████╗░",
	"╚════██╗",
	"░█████╔╝",
	"░╚═══██╗",
	"██████╔╝",
	"╚═════╝░",
}

var ASCII_GAME_OVER = []string{
	"░██████╗░░█████╗░███╗░░░███╗███████╗  ░█████╗░██╗░░░██╗███████╗██████╗░",
	"██╔════╝░██╔══██╗████╗░████║██╔════╝  ██╔══██╗██║░░░██║██╔════╝██╔══██╗",
	"██║░░██╗░███████║██╔████╔██║█████╗░░  ██║░░██║╚██╗░██╔╝█████╗░░██████╔╝",
	"██║░░╚██╗██╔══██║██║╚██╔╝██║██╔══╝░░  ██║░░██║░╚████╔╝░██╔══╝░░██╔══██╗",
	"╚██████╔╝██║░░██║██║░╚═╝░██║███████╗  ╚█████╔╝░░╚██╔╝░░███████╗██║░░██║",
	"░╚═════╝░╚═╝░░╚═╝╚═╝░░░░░╚═╝╚══════╝  ░╚════╝░░░░╚═╝░░░╚══════╝╚═╝░░╚═╝",
}

var ASCII_YOU_WON = []string{
	"██╗░░░██╗░█████╗░██╗░░░██╗  ░██╗░░░░░░░██╗░█████╗░███╗░░██╗",
	"╚██╗░██╔╝██╔══██╗██║░░░██║  ░██║░░██╗░░██║██╔══██╗████╗░██║",
	"░╚████╔╝░██║░░██║██║░░░██║  ░╚██╗████╗██╔╝██║░░██║██╔██╗██║",
	"░░╚██╔╝░░██║░░██║██║░░░██║  ░░████╔═████║░██║░░██║██║╚████║",
	"░░░██║░░░╚█████╔╝╚██████╔╝  ░░╚██╔╝░╚██╔╝░╚█████╔╝██║░╚███║",
	"░░░╚═╝░░░░╚════╝░░╚═════╝░  ░░░╚═╝░░░╚═╝░░░╚════╝░╚═╝░░╚══╝",
}

var returnToLobbyText = "Press [Enter] to return to lobby"

var TRON_COLORS = [8]string{"blue", "red", "green", "purple", "yellow", "orange", "white", "teal"}

type TronDirection int64

const (
	TronUp TronDirection = iota
	TronRight
	TronDown
	TronLeft
)

type Position struct {
	X int
	Y int
}

var mu sync.Mutex

type TronClientState struct {
	Timestep  int
	Alive     bool
	Color     string
	X         int
	Y         int
	Direction TronDirection
	PlayerNum int
}

type TronGameState struct {
	Width            int
	Height           int
	Ended            bool
	Winner           string
	Collisions       []byte
	ClientStates     map[string]TronClientState
	CommitedTimeStep int
}

// use this to coalesce optimistic game state
// in the future, generalize this
// var localCollisions []byte

type TronCommandType int64

const (
	TronMoveCmd TronCommandType = iota
	TronDeadCmd
)

type TronCommand struct {
	Id        string
	Type      TronCommandType
	Timestep  int
	PlayerID  string
	Direction TronDirection
}

func (tc TronCommand) String() string {
	var dir string
	switch tc.Direction {
	case TronDown:
		dir = "TronDown"
	case TronUp:
		dir = "TronUp"
	case TronLeft:
		dir = "TronLeft"
	case TronRight:
		dir = "TronRight"
	}
	return fmt.Sprintf("[CMD]%s: %s]", tc.PlayerID[:3], dir)
}

// BASIC QUEUE IMPLEMENTATION

type BasicQueue[T any] []T

func (bq *BasicQueue[T]) initialize(queue *[]T) {
	*bq = *queue
}

func (bq *BasicQueue[T]) push(element ...T) {
	*bq = append(*bq, element...)
}

func (bq *BasicQueue[T]) pop() (T, bool) {
	if len(*bq) == 0 {
		var nullT T
		return nullT, false
	}
	element := (*bq)[0] // The first element is the one to be dedequeued.
	*bq = (*bq)[1:]
	return element, true
}

/*

P1 50 Left
P2 50 Right
P1 56 Up
P2 53 Left


=== P2 ===
TIMESTEP: 54

uncommited logs:
P1 50 Left
P2 50 Right
P1 56 Up <-- just arrived
P2 57 Left
P2 58 Up

client pred:
P2 53 Left
P2 54 Up
P2 57 Down


If higher timestep comes in, shift everything up so that timestep matches, relative

uncommited logs:
P1 56 Up <-- just arrived

client pred:
P2 53 Left --> P2 56 Left
P2 54 Up --> P2 57 Up

NEW TIMESTEP: 57


maybe try relative timesteps to uncommitted logs?

reset client prediction at every appendEntries?
Send an id with each own player move, maintain client prediction up until that move
*/

type TronGameView struct {
	View
	Game[TronGameState, TronClientState]
	CommitedGameState TronGameState
	WorkingGameState  TronGameState
	MoveQueue         []TronCommand
	LatestInputDir    TronDirection
}

const CLIENT_LAG_TIMESTEP = 0
const FRAGMENTS = 2

func NewTronGameView(lobby *Lobby) *TronGameView {
	return &TronGameView{
		Game: Game[TronGameState, TronClientState]{
			// ID is the lobby ID, not the player
			ID:             lobby.ID,
			PlayerIDs:      lobby.PlayerIDs,
			Name:           lobby.Name,
			Me:             arcade.Server.ID,
			HostID:         lobby.HostID,
			HostSyncPeriod: 2000,
			TimestepPeriod: 100,
			Timestep:       0,
		},
	}
}

var lastReceivedInp = make(map[string]int)
var needToProcessInput = false

var currGameUpdateId = ""
var currCollisions []byte
var currFragments = 0

var showCommits = false

const (
	TronInitScreen = iota
	TronGameScreen
	TronWinScreen
)

var gameRenderState = TronInitScreen
var countdownNum = 3

/*
1. Initialize game state
2. On every TIMESTEP:
  a0. increment TIMESTEP
	a. Calculate self state and send command, add to moveQ
	b. Sleep
	c. Ingest state from raft log:
		1. replay uncomitted logs ontop of stored game state
			a. iterate, keep track of current latest_timestep
			b. any timesteps < latest_timestep will be replayed on top of latest_timestep, w.r.t. relative timing
			c. if final latest_timestep > TIMESTEP, set TIMESTEP = latest_timestep (not counting "replayed" timesteps)
		2. client predict up until current timestep
		3. client predict self based on moveQ, replaying on top of latest_timestep
3. When applyMsg is received, directly modify the base game state

*/
func (tg *TronGameView) Init() {

	// JANK
	var me int
	for i := range tg.PlayerIDs {
		if tg.PlayerIDs[i] == tg.Me {
			me = i
		}
	}
	applyChan := make(chan raft.ApplyMsg)

	clients := []*net.Client{}
	for _, playerId := range tg.PlayerIDs {
		if playerId == tg.Me {
			myClient := net.Client{}
			clients = append(clients, &myClient)
		}
		if client, ok := arcade.Server.Network.GetClient(playerId); ok {
			clients = append(clients, client)
		}
	}

	// JANK
	// fmt.Println("CLIENTS: ", clients)
	tg.RaftServer = raft.Make(clients, me, applyChan, arcade.Server.Network)

	width, height := arcade.ViewManager.screen.displaySize()

	clientStates := make(map[string]TronClientState)
	startingPos, startingDir := getStartingPosAndDir()

	for i, playerID := range tg.PlayerIDs {
		x := startingPos[i][0]
		y := startingPos[i][1]
		clientStates[playerID] = TronClientState{tg.Timestep, true, TRON_COLORS[i], x, y, startingDir[i], i}
		lastReceivedInp[playerID] = 0
	}

	// fmt.Print(clientStates)

	tg.CommitedGameState = TronGameState{width, height, false, "", initCollisions(), clientStates, 0}
	tg.WorkingGameState = TronGameState{width, height, false, "", initCollisions(), clientStates, 0}

	tg.start()

	go func() {
		for i := 1; i > 0; i-- {
			countdownNum = i
			arcade.ViewManager.RequestRender()
			time.Sleep(time.Duration(int(time.Second)))
		}

		gameRenderState = TronGameScreen
		for !tg.WorkingGameState.Ended {
			tg.Timestep += 1
			tg.updateSelf()
			arcade.ViewManager.RequestRender()
			time.Sleep(time.Duration(tg.TimestepPeriod * int(time.Millisecond)))
			tg.updateWorkingGameState()
			arcade.ViewManager.RequestRender()
			// mu.Lock()
			// if tg.Me == tg.HostID {
			// 	if ended, winner := tg.shouldWin(); ended {
			// 		tg.GameState.Ended = ended
			// 		tg.GameState.Winner = winner
			// 		tg.sendGameUpdate()
			// 		tg.sendEndGame(winner)
			// 	}
			// }
			// mu.Unlock()
		}

		gameRenderState = TronWinScreen
		arcade.ViewManager.RequestRender()
	}()

	if tg.Me == tg.HostID && tg.HostSyncPeriod > 0 {
		go tg.startHostSync()
	}

}

func (tg *TronGameView) startHostSync() {
	// for tg.Started {
	// 	time.Sleep(time.Duration(tg.HostSyncPeriod * int(time.Millisecond)))
	// 	tg.commitGameState()
	// 	tg.sendGameUpdate()
	// }
}

func (tg *TronGameView) ProcessEvent(ev interface{}) {
	switch ev := ev.(type) {
	case *ClientDisconnectEvent:
		// process disconnected client
	case *tcell.EventKey:
		if ev.Key() == tcell.KeyEnter {
			mu.Lock()
			gamestate := tg.WorkingGameState.Ended
			mu.Unlock()
			if gamestate {
				// arcade.Lobby.mu.RLock()
				// hostID := arcade.Lobby.HostID
				// lobbyID := arcade.Lobby.ID
				// arcade.Lobby.mu.RUnlock()

				// if arcade.Server.ID == hostID {
				// arcade.lobbyMux.Lock()
				// arcade.Lobby = &Lobby{}
				// arcade.lobbyMux.Unlock()

				arcade.Server.EndAllHeartbeats()
				// send updates to everyone

				// arcade.Server.Network.ClientsRange(func(client *Client) bool {
				// 	if client.Distributor {
				// 		return true
				// 	}

				// 	arcade.Server.Network.Send(client, NewLobbyEndMessage(lobbyID))

				// 	return true
				// })

				arcade.ViewManager.SetView(NewGamesListView())

				// }
			}
			return
		}
		tg.ProcessEventKey(ev)
	}
}

func (tg *TronGameView) ProcessEventKey(ev *tcell.EventKey) {
	if needToProcessInput {
		return
	}
	key := ev.Key()

	clientState := tg.getMyState()
	switch key {
	case tcell.KeyUp:
		if clientState.Direction != TronDown {
			tg.LatestInputDir = TronUp
			needToProcessInput = true
		}
	case tcell.KeyRight:
		if clientState.Direction != TronLeft {
			tg.LatestInputDir = TronRight
			needToProcessInput = true
		}
	case tcell.KeyDown:
		if clientState.Direction != TronUp {
			tg.LatestInputDir = TronDown
			needToProcessInput = true
		}
	case tcell.KeyLeft:
		if clientState.Direction != TronRight {
			tg.LatestInputDir = TronLeft
			needToProcessInput = true
		}
	case tcell.KeyCtrlG:
		showCommits = !showCommits

	}

}

func (tg *TronGameView) ProcessMessage(from *net.Client, p interface{}) interface{} {
	switch p := p.(type) {
	// case GameUpdateMessage[TronGameState, TronClientState]:
	// 	tg.handleGameUpdate(p)
	// case ClientUpdateMessage[TronClientState]:
	// 	tg.handleClientUpdate(p)
	case EndGameMessage:
		tg.handleEndGame(p)
	}
	return nil
}

func (tg *TronGameView) Render(s *Screen) {
	s.ClearContent()

	displayWidth, displayHeight := arcade.ViewManager.screen.displaySize()
	boxStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorTeal)
	s.DrawBox(1, 1, displayWidth-2, displayHeight-2, boxStyle, false)

	switch gameRenderState {
	case TronInitScreen:
		myState := tg.getMyState()
		style := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorNames[myState.Color])
		chr := getDirChr(myState.Direction)
		s.DrawText(myState.X, myState.Y, style, chr)

		// draw countdown
		var countdownAscii []string
		switch countdownNum {
		case 1:
			countdownAscii = COUNTDOWN_1
		case 2:
			countdownAscii = COUNTDOWN_2
		case 3:
			countdownAscii = COUNTDOWN_3
		}

		headerY := (displayHeight - len(countdownAscii)) / 2
		headerX := (displayWidth - utf8.RuneCountInString(countdownAscii[0])) / 2
		for i := range countdownAscii {
			s.DrawText(headerX, i+headerY, boxStyle, countdownAscii[i])
		}

	case TronGameScreen:
		tg.renderGame(s)
	case TronWinScreen:
		tg.renderGame(s)

		// draw countdown
		endGameText := ASCII_GAME_OVER
		if tg.WorkingGameState.Winner == tg.Me {
			endGameText = ASCII_YOU_WON
		}

		headerY := (displayHeight - len(endGameText)) / 2
		headerX := (displayWidth - utf8.RuneCountInString(endGameText[0])) / 2
		for i := range endGameText {
			s.DrawText(headerX, i+headerY, boxStyle, endGameText[i])
		}

		s.DrawText((displayWidth-utf8.RuneCountInString(returnToLobbyText))/2, displayHeight-6, boxStyle, returnToLobbyText)

	}

}

func (tg *TronGameView) renderGame(s *Screen) {
	for row := 0; row < tg.WorkingGameState.Width; row++ {
		for col := 0; col < tg.WorkingGameState.Height; col++ {
			if ok, playerNum := tg.getCollision(tg.WorkingGameState.Collisions, row, col); ok && playerNum >= 0 {
				style := tcell.StyleDefault.Background(tcell.ColorNames[TRON_COLORS[playerNum]])
				s.DrawText(row, col, style, " ")
			}

			if showCommits {
				if ok, playerNum := tg.getCollision(tg.WorkingGameState.Collisions, row, col); ok && playerNum >= 0 && playerNum < len(TRON_COLORS)-1 {
					style := tcell.StyleDefault.Background(tcell.ColorNames[TRON_COLORS[playerNum+1]])
					s.DrawText(row, col, style, " ")
				}
			}
		}
	}

	for _, client := range tg.WorkingGameState.ClientStates {
		if client.Alive {
			style := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorNames[client.Color])
			chr := getDirChr(client.Direction)
			s.DrawText(client.X, client.Y, style, chr)
			if client.Direction == TronLeft {
				s.DrawText(client.X+1, client.Y, style, " ")
			} else if client.Direction == TronRight {
				s.DrawText(client.X-1, client.Y, style, " ")
			}
		} else {
			style := tcell.StyleDefault.Foreground(tcell.ColorNames[client.Color])
			s.DrawText(client.X, client.Y, style, "😵")
		}
	}
}

func (tg *TronGameView) updateState() {

}

func (tg *TronGameView) updateSelf() {
	mu.Lock()
	defer mu.Unlock()
	if needToProcessInput {

		cmd := TronCommand{uuid.NewString(), TronMoveCmd, tg.Timestep, tg.Me, tg.LatestInputDir}

		tg.RaftServer.Start(cmd)
		tg.MoveQueue = append(tg.MoveQueue, cmd)
		needToProcessInput = false
	}

}

func (tg *TronGameView) updateWorkingGameState() {
	raftLog, _ := tg.RaftServer.GetLog()
	entries := raftLog.GetEntries()

	convertedEntries := []TronCommand{}
	for _, entry := range entries {
		if cmd, ok := readLogEntryAsTronCmd(entry.Command); ok {
			convertedEntries = append(convertedEntries, cmd)
		}
	}

	// log.Println("[RAFT]:", "entries on", tg.Me, convertedEntries)

	workingGameState := TronGameState{}
	copier.CopyWithOption(&workingGameState, &tg.CommitedGameState, copier.Option{DeepCopy: true})

	// process badly ordered cmd logs
	latestTimestep := -1
	lastDelayedTimeByPlayer := make(map[string]int)
	var processedLogs BasicQueue[TronCommand]

	for _, entry := range entries {

		cmd, ok := readLogEntryAsTronCmd(entry.Command)
		if !ok {
			log.Println("[RAFT]:", "FAILED TO PARSE ENTRY", entry.Command)
			continue
		}

		if cmd.Timestep > latestTimestep {
			latestTimestep = cmd.Timestep
			lastDelayedTimeByPlayer = make(map[string]int)
		}

		if cmd.Timestep < latestTimestep {
			ogTimestep := cmd.Timestep
			if lastTime, ok := lastDelayedTimeByPlayer[cmd.PlayerID]; ok {
				cmd.Timestep = int(math.Max(float64(latestTimestep), float64(latestTimestep+cmd.Timestep-lastTime)))
			} else {
				cmd.Timestep = latestTimestep
			}
			lastDelayedTimeByPlayer[cmd.PlayerID] = ogTimestep
		}
		processedLogs.push(cmd)

		tg.truncateMoveQueueIfNecessary(cmd)
	}

	if latestTimestep > tg.Timestep {
		tg.Timestep = latestTimestep
	}

	// JANK: mixing in move queue in to processed logs instead of replaying on top, could cause jumps
	processedLogs.push(tg.MoveQueue...)

	sort.Slice(processedLogs, func(i, j int) bool {
		return processedLogs[i].Timestep < processedLogs[j].Timestep
	})

	currentTimestep := workingGameState.CommitedTimeStep + 1
	// replay cmds on top of gamestate
	for len(processedLogs) > 0 || currentTimestep <= tg.Timestep {
		if len(processedLogs) > 0 && processedLogs[0].Timestep == currentTimestep {
			// fmt.Print("HAS PROCESSED LOGS AHHH")
			if cmd, ok := processedLogs.pop(); ok {
				clientState := workingGameState.ClientStates[cmd.PlayerID]

				switch cmd.Type {
				case TronMoveCmd:
					clientState.Direction = cmd.Direction
				}

				workingGameState.ClientStates[cmd.PlayerID] = clientState
				// fmt.Printf("A: %d %d\n", cmd.Direction, workingGameState.ClientStates[tg.Me].Direction)
			}
		} else {
			// apply prev timestep state change
			// fmt.Printf("B:%d\n ", workingGameState.ClientStates[tg.Me].Direction)
			workingGameState = tg.clientPredict(workingGameState, 1)
			currentTimestep += 1
		}
	}
	// fmt.Print("after: ", workingGameState.ClientStates)
	tg.WorkingGameState = workingGameState
}

// blindly truncates move queue if id matches. Could potentially cut out earlier cmds in the moveQueue
func (tg *TronGameView) truncateMoveQueueIfNecessary(cmd TronCommand) {
	mu.Lock()
	defer mu.Unlock()
	for i, move := range tg.MoveQueue {
		if move.Id == cmd.Id {
			tg.MoveQueue = tg.MoveQueue[i+1:]
			return
		}
	}
}

func (tg *TronGameView) clientPredict(gameState TronGameState, numTimesteps int) TronGameState {
	for i := 0; i < numTimesteps; i++ {
		for playerId, clientState := range gameState.ClientStates {
			if !clientState.Alive {
				continue
			}

			gameState.Collisions = tg.setCollision(gameState.Collisions, clientState.X, clientState.Y, clientState.PlayerNum)

			newX := clientState.X
			newY := clientState.Y

			// fmt.Printf("C: %d\n", gameState.ClientStates[tg.Me].Direction)

			switch clientState.Direction {
			case TronUp:
				newY -= 1
			case TronRight:
				newX += 1
			case TronDown:
				newY += 1
			case TronLeft:
				newX -= 1
			}

			clientState.X = newX
			clientState.Y = newY

			gameState.ClientStates[playerId] = clientState
		}

		// can def optimize out this 2nd loop
		for playerId, clientState := range gameState.ClientStates {
			if tg.shouldDie(clientState, gameState) {
				gameState.ClientStates[playerId] = tg.die(clientState)
			}
		}
	}
	return gameState
}

// func (tg *TronGameView) handleGameUpdate(data GameUpdateMessage[TronGameState, TronClientState]) {
// 	mu.Lock()
// 	defer mu.Unlock()
// 	if data.ID != currGameUpdateId {
// 		currGameUpdateId = data.ID
// 		currFragments = 0
// 		currCollisions = initCollisions()
// 	}

// 	gameState := data.GameUpdate
// 	size := len(currCollisions)
// 	currCollisions = append(append(currCollisions[:data.FragmentNum*size/FRAGMENTS], gameState.Collisions...), currCollisions[int(math.Min(float64(size), float64((data.FragmentNum+1)*size/FRAGMENTS))):]...)
// 	currFragments += 1

// 	if currFragments != FRAGMENTS {
// 		return
// 	}
// 	gameState.Collisions = currCollisions
// 	tg.GameState = gameState

// 	for id, lastInp := range data.LastInps {
// 		currClient := tg.ClientStates[id]
// 		if lastInp >= currClient.CommitTimestep {
// 			diff := lastInp - currClient.CommitTimestep

// 			currClient.CommitTimestep = lastInp
// 			currClient.PathX = currClient.PathX[int(math.Min(float64(diff), float64(len(currClient.PathX)))):]
// 			currClient.PathY = currClient.PathY[int(math.Min(float64(diff), float64(len(currClient.PathY)))):]
// 			if lastReceivedInp[id] < lastInp {
// 				lastReceivedInp[id] = lastInp
// 			}
// 			tg.ClientStates[id] = currClient
// 		} else {
// 			panic("incoming client < currclient commitTimestep")
// 		}
// 	}
// 	tg.recalculateCollisions()
// }

// func (tg *TronGameView) handleClientUpdate(data ClientUpdateMessage[TronClientState]) {
// 	mu.Lock()
// 	defer mu.Unlock()
// 	update := data.Update
// 	state := tg.ClientStates[data.Id]

// 	if state.CommitTimestep <= update.CommitTimestep {
// 		if update.CommitTimestep <= state.Timestep+1 {
// 			diff := update.CommitTimestep - state.CommitTimestep
// 			update.PathX = append(state.PathX[:diff], update.PathX...)
// 			update.PathY = append(state.PathY[:diff], update.PathY...)
// 		} else {
// 			update = tg.clientPredict(state, update.CommitTimestep-1) // client predict to fill gap
// 			update.PathX = append(state.PathX, update.PathX...)
// 			update.PathY = append(state.PathY, update.PathY...)
// 		}
// 	} else {
// 		if update.Timestep >= state.CommitTimestep {
// 			diff := state.CommitTimestep - update.CommitTimestep
// 			update.PathX = update.PathX[diff:]
// 			update.PathY = update.PathY[diff:]
// 		} else {
// 			panic("update out of date: " + fmt.Sprintf("%d [%d] < %d [%d]", update.CommitTimestep, len(update.PathX), state.CommitTimestep, len(state.PathX)))
// 		}
// 	}

// 	update.CommitTimestep = state.CommitTimestep

// 	tg.ClientStates[data.Id] = update
// 	arcade.ViewManager.RequestRender()
// 	tg.recalculateCollisions()

// 	lastReceivedInp[data.Id] = update.Timestep
// }

func (tg *TronGameView) handleEndGame(data EndGameMessage) {
	tg.WorkingGameState.Ended = true
	tg.WorkingGameState.Winner = data.Winner
	arcade.ViewManager.RequestRender()
}

// func (tg *TronGameView) commitGameState() {
// 	mu.Lock()
// 	defer mu.Unlock()
// 	collisions := tg.GameState.Collisions
// 	for id, player := range tg.ClientStates {
// 		for i := 0; i < lastReceivedInp[id]-player.CommitTimestep; i++ {
// 			collisions = tg.setCollision(collisions, player.PathX[i], player.PathY[i], player.PlayerNum)
// 		}

// 		diff := lastReceivedInp[id] - player.CommitTimestep
// 		player.CommitTimestep = lastReceivedInp[id]
// 		player.PathX = player.PathX[int(math.Min(float64(diff), float64(len(player.PathX)))):]
// 		player.PathY = player.PathY[int(math.Min(float64(diff), float64(len(player.PathY)))):]

// 		tg.ClientStates[id] = player
// 	}
// 	tg.GameState.Collisions = collisions
// 	tg.recalculateCollisions()
// }

func (tg *TronGameView) sendEndGame(winner string) {
	endGame := &EndGameMessage{Message: message.Message{Type: "end_game"}, Winner: winner}

	for clientId := range tg.WorkingGameState.ClientStates {
		if client, ok := arcade.Server.Network.GetClient(clientId); ok && clientId != tg.Me {
			go func() {
				var err error = nil
				for err == nil {
					_, err = arcade.Server.Network.SendAndReceive(client, endGame)
				}
			}()
		}
	}
}

func (g *Game[GS, CS]) sendClientUpdate(update CS) {
	// clientUpdate := &ClientUpdateMessage[CS]{Message: Message{Type: "client_update"}, Id: g.Me, Update: update}

	// for clientId := range g.ClientStates {
	// 	if client, ok := arcade.Server.Network.GetClient(clientId); ok && clientId != g.Me {
	// 		arcade.Server.Network.Send(client, clientUpdate)
	// 	}
	// }
}

// func (tg *TronGameView) sendGameUpdate() {
// 	if tg.Me != tg.HostID {
// 		return
// 	}

// 	arcade.Server.RLock()
// 	defer arcade.Server.RUnlock()

// 	// data := &GameUpdateMessage[TronGameState, TronClientState]{Message{Type: "game_update"}, gameState, lastReceivedInp, id, i}

// 	for clientId := range tg.ClientStates {
// 		if client, ok := arcade.Server.Network.GetClient(clientId); ok && clientId != tg.Me {
// 			id := uuid.NewString()
// 			for i := 0; i < FRAGMENTS; i++ {
// 				gameState := tg.GameState
// 				size := len(gameState.Collisions)
// 				frag := gameState.Collisions[i*size/FRAGMENTS : int(math.Min(float64(size), float64((i+1)*size/FRAGMENTS)))]
// 				gameState.Collisions = frag
// 				data := &GameUpdateMessage[TronGameState, TronClientState]{message.Message{Type: "game_update"}, gameState, lastReceivedInp, id, i}
// 				arcade.Server.Network.Send(client, data)
// 			}
// 		}
// 	}
// }

// func (tg *TronGameView) recalculateCollisions() {
// 	copy(localCollisions, tg.GameState.Collisions)
// 	for _, player := range tg.ClientStates {
// 		for i := 0; i < len(player.PathX); i++ {
// 			localCollisions = tg.setCollision(localCollisions, player.PathX[i], player.PathY[i], player.PlayerNum)
// 		}
// 	}
// }

// GAME FUNCTIONS
func getStartingPosAndDir() ([][2]int, []TronDirection) {
	width, height := arcade.ViewManager.screen.displaySize()
	width -= 1 // account for tron border
	height -= 1
	margin := int(math.Round(math.Min(float64(width)/8, float64(height)/8)))
	return [][2]int{{margin, margin}, {width - margin, height - margin}, {width - margin, margin}, {margin, height - margin}, {width / 2, margin}, {width - margin, height / 2}, {width / 2, height - margin}, {margin, height / 2}}, []TronDirection{TronRight, TronLeft, TronDown, TronUp, TronDown, TronLeft, TronUp, TronRight}
}

func (tg *TronGameView) shouldDie(player TronClientState, gameState TronGameState) bool {
	collides, _ := tg.getCollision(gameState.Collisions, player.X, player.Y)
	return tg.isOutOfBounds(player.X, player.Y) || collides
}

func (tg *TronGameView) die(player TronClientState) TronClientState {
	player.Alive = false
	return player
}

func (tg *TronGameView) shouldWin() (bool, string) {
	winner := ""
	if len(tg.WorkingGameState.ClientStates) == 1 {
		winner = "can't win without friends :^)"
	}

	for id, client := range tg.WorkingGameState.ClientStates {
		if client.Alive {
			if winner != "" {
				return false, ""
			}
			winner = id
		}
	}
	return true, winner
}

func (tg *TronGameView) isOutOfBounds(x int, y int) bool {
	return x <= 1 || x >= tg.WorkingGameState.Width-2 || y <= 1 || y >= tg.WorkingGameState.Height-2
}

func (tg *TronGameView) setCollision(collisions []byte, x int, y int, playerNum int) []byte {
	width, _ := arcade.ViewManager.screen.displaySize()
	if !tg.isOutOfBounds(x, y) && playerNum < 8 {
		ind := y*width + x
		collisions[ind/2] |= byte(playerNum<<1+1) << ((ind % 2) * 4)
	}
	return collisions
}

// returns bool of collision and player num
func (tg *TronGameView) getCollision(collisions []byte, x int, y int) (bool, int) {
	width, _ := arcade.ViewManager.screen.displaySize()
	if !tg.isOutOfBounds(x, y) {
		ind := y*width + x
		offset := ((ind % 2) * 4)
		coll := collisions[ind/2] >> offset

		if coll&1 == 1 {
			return true, int((coll >> 1) & 7)
		} else {
			return false, -1
		}
	}
	return true, -1
}

func (tg *TronGameView) getMyState() TronClientState {
	return tg.WorkingGameState.ClientStates[tg.Me]
}

func (tg *TronGameView) setMyState(state TronClientState) {
	tg.WorkingGameState.ClientStates[tg.Me] = state
}

func getDirChr(dir TronDirection) string {
	switch dir {
	case TronUp:
		return "▲"
	case TronRight:
		return "▶"
	case TronDown:
		return "▼"
	case TronLeft:
		return "◀"
	}
	return "?"
}

func toByteStr(collisions [][]bool) string {
	width, height := arcade.ViewManager.screen.displaySize()
	totalSize := width * height

	bytes := make([]byte, int(math.Ceil(float64(totalSize)/8)))
	for i := range collisions {
		for j := range collisions[i] {
			byteInd := (i + j) / 8
			if collisions[i][j] {
				bytes[byteInd] |= 1 << ((i + j) % 8)
			}
		}
	}
	return string(bytes)
}

func fromBytestr(byteStr string) [][]bool {
	width, height := arcade.ViewManager.screen.displaySize()
	// totalSize := width * height

	collisions := make([][]bool, width)
	for i := range collisions {
		collisions[i] = make([]bool, height)
	}

	bytes := []byte(byteStr)
	for byteInd, b := range bytes {
		ind := byteInd * 8
		for x := 0; x < 8; x++ {
			if b>>x&1 == 1 {
				i := (ind + x) / width
				j := (ind + x) - i*width
				collisions[i][j] = true
			}
		}
	}
	return collisions
}

func initCollisions() []byte {
	width, height := arcade.ViewManager.screen.displaySize()
	return make([]byte, int(math.Ceil(float64(width*height)/2)))
}

func (v *TronGameView) GetHeartbeatMetadata() encoding.BinaryMarshaler {
	return nil
}

func (v *TronGameView) Unload() {
}

func readLogEntryAsTronCmd(entry interface{}) (TronCommand, bool) {
	var cmd TronCommand

	if jsonStr, err := json.Marshal(entry); err == nil {
		if err := json.Unmarshal(jsonStr, &cmd); err == nil {
			return cmd, true
		}
	}
	return cmd, false
}
