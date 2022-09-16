package arcade

import (
	"encoding"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"

	"arcade/arcade/net"
	"arcade/raft"

	"github.com/gdamore/tcell/v2"
	"github.com/google/uuid"
	"github.com/jinzhu/copier"
)

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

var mu sync.RWMutex

var c = sync.NewCond(&mu)

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

type TronCommandType int64

const (
	TronMoveCmd TronCommandType = iota
	TronEndGameCmd
)

type TronCommand struct {
	Id        string
	Type      TronCommandType
	Timestep  int
	PlayerID  string
	Direction TronDirection
	Winner    string
}

func (tc TronCommand) String() string {
	if tc.Type == TronEndGameCmd {
		return fmt.Sprintf("%s[%s, W:%s]", tc.Id[:int(math.Min(3, float64(len(tc.Id))))], tc.PlayerID[:int(math.Min(3, float64(len(tc.PlayerID))))], tc.Winner[:int(math.Min(3, float64(len(tc.Winner))))])
	}
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
	return fmt.Sprintf("%s[%d,%s, %s]", tc.Id[:int(math.Min(3, float64(len(tc.Id))))], tc.Timestep, tc.PlayerID[:int(math.Min(3, float64(len(tc.PlayerID))))], dir)
}

// BASIC QUEUE IMPLEMENTATION

type BasicQueue[T any] []T

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
	mgr *ViewManager
	Game[TronGameState, TronClientState]
	CommitedGameState TronGameState
	WorkingGameState  TronGameState
	MoveQueue         []TronCommand
	NextDir           TronDirection
	LatestInputDir    TronDirection
	ApplyChan         chan raft.ApplyMsg
	lastApplyMsgInd   int
	gameRenderState   TronGameRenderState
	lobby             *Lobby
}

const CLIENT_LAG_TIMESTEP = 0
const FRAGMENTS = 2

func NewTronGameView(mgr *ViewManager, lobby *Lobby) *TronGameView {
	return &TronGameView{
		mgr: mgr,
		Game: Game[TronGameState, TronClientState]{
			// ID is the lobby ID, not the player
			ID:             lobby.ID,
			PlayerIDs:      lobby.PlayerIDs,
			Name:           lobby.Name,
			Me:             arcade.Server.ID,
			HostID:         lobby.HostID,
			HostSyncPeriod: 2000,
			TimestepPeriod: 80,
			Timestep:       0,
		},
		lobby: lobby,
	}
}

var lastReceivedInp = make(map[string]int)
var needToProcessInput = false

var showCommits = false

type TronGameRenderState int64

const (
	TronInitScreen TronGameRenderState = iota
	TronGameScreen
	TronWinScreen
)

// var gameRenderState = TronInitScreen
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

	mu.Lock()
	// JANK
	var me int
	for i := range tg.PlayerIDs {
		if tg.PlayerIDs[i] == tg.Me {
			me = i
		}
	}
	tg.ApplyChan = make(chan raft.ApplyMsg)

	clients := []*net.Client{}
	for _, playerId := range tg.PlayerIDs {
		if playerId == tg.Me {
			myClient := net.Client{}
			clients = append(clients, &myClient)
		} else if client, ok := arcade.Server.Network.GetClient(playerId); ok {
			clients = append(clients, client)
		}
	}

	// JANK
	tg.RaftServer = raft.Make(clients, me, tg.ApplyChan, arcade.Server.Network, tg.TimestepPeriod, c)

	log.Println("RAFT SERVER:", &tg.RaftServer)

	width, height := tg.mgr.screen.displaySize()

	clientStates := make(map[string]TronClientState)
	startingPos, startingDir := tg.getStartingPosAndDir()

	for i, playerID := range tg.PlayerIDs {
		x := startingPos[i][0]
		y := startingPos[i][1]
		clientStates[playerID] = TronClientState{tg.getTimestep(), true, TRON_COLORS[i], x, y, startingDir[i], i}
		lastReceivedInp[playerID] = 0

		if playerID == tg.Me {
			tg.LatestInputDir = startingDir[i]
		}
	}

	tg.NextDir = -1

	tg.CommitedGameState = TronGameState{width, height, false, "", tg.initCollisions(), clientStates, -1}
	tg.WorkingGameState = TronGameState{width, height, false, "", tg.initCollisions(), clientStates, -1}
	mu.Unlock()
	tg.startApplyChanHandler()

	go func() {

		for i := 3; i > 0; i-- {
			countdownNum = i
			mu.RLock()
			tg.mgr.RequestRender()
			mu.RUnlock()
			time.Sleep(time.Duration(int(time.Second)))
		}

		mu.Lock()
		tg.RaftServer.StartTime()

		tg.gameRenderState = TronGameScreen
		lastTimestep := -1
		for !tg.CommitedGameState.Ended {
			c.Wait()

			timestep := tg.RaftServer.GetTimestep()
			if timestep == lastTimestep {
				panic(fmt.Sprintf("SAME TIMESTEP, %d", timestep))
			} else {
				lastTimestep = timestep
			}

			// update gamestate and render for previous timestep
			tg.updateWorkingGameState(timestep - 1)

			tg.mgr.RequestRender()

			// DEBUG MODE
			tg.mgr.RLock()
			if tg.mgr.showDebug {
				w, _ := tg.mgr.screen.displaySize()
				style := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
				_, isLeader := tg.RaftServer.GetState()
				tg.mgr.screen.DrawText(w+1, 0, style, fmt.Sprintf("L:%t, T:%d", isLeader, timestep))
			}
			tg.mgr.RUnlock()

			// send command for current timestep
			tg.updateSelf()
			tg.WorkingGameState = tg.clientPredict(tg.WorkingGameState, 1, []string{tg.Me})
			tg.mgr.RequestRender()

		}

		tg.gameRenderState = TronWinScreen
		mu.Unlock()

		tg.mgr.RequestRender()
	}()

}

func (tg *TronGameView) ProcessEvent(ev interface{}) {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		if ev.Key() == tcell.KeyEnter {
			mu.RLock()
			if tg.CommitedGameState.Ended {
				// tg.RaftServer.Kill()
				// arcade.Server.EndAllHeartbeats()
				// lobby := tg.lobby
				mu.RUnlock()
				tg.mgr.SetView(NewGamesListView(tg.mgr)) //TODO: change this to the lobby view
			}

			return
		}
		tg.ProcessEventKey(ev)
	}
}

func (tg *TronGameView) ProcessEventKey(ev *tcell.EventKey) {

	key := ev.Key()
	mu.RLock()
	clientState := tg.getMyState()
	mu.RUnlock()
	var newDir TronDirection

	switch key {
	case tcell.KeyCtrlG:
		showCommits = !showCommits
		return
	case tcell.KeyUp:
		newDir = TronUp
	case tcell.KeyRight:
		newDir = TronRight
	case tcell.KeyDown:
		newDir = TronDown
	case tcell.KeyLeft:
		newDir = TronLeft
	}

	mu.Lock()
	defer mu.Unlock()

	if needToProcessInput {
		// TODO: check for tron direction here as well and don't send cmd if same dir
		if canMoveInDir(tg.LatestInputDir, newDir) {
			log.Println("setting Nextdir", newDir)
			tg.NextDir = newDir
		}
	} else {
		if canMoveInDir(clientState.Direction, newDir) {
			tg.NextDir = -1
			tg.LatestInputDir = newDir
			needToProcessInput = true
		}

	}

}

func (tg *TronGameView) ProcessMessage(from *net.Client, p interface{}) interface{} {
	return tg.RaftServer.ProcessMessage(from, p)
}

func (tg *TronGameView) Render(s *Screen) {
	// mu.Lock()
	// defer mu.Unlock()

	s.ClearContent()

	displayWidth, displayHeight := tg.mgr.screen.displaySize()
	boxStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorTeal)
	s.DrawBox(1, 1, displayWidth-2, displayHeight-2, boxStyle, false)

	switch tg.gameRenderState {
	case TronInitScreen:
		myState := tg.getMyState()
		style := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorNames[myState.Color])
		chr := getDirChr(myState.Direction)
		s.DrawText(myState.X, myState.Y, style, chr)

		// draw countdown
		s.DrawBlockText(CenterX, CenterY, boxStyle, strconv.Itoa(countdownNum), true)
	case TronGameScreen:
		tg.renderGame(s)
	case TronWinScreen:
		tg.renderGame(s)

		if tg.WorkingGameState.Winner == tg.Me {
			s.DrawBlockText(CenterX, CenterY, boxStyle, "YOU WON", true)
		} else {
			s.DrawBlockText(CenterX, CenterY, boxStyle, "GAME OVER", true)
		}

		s.DrawText((displayWidth-utf8.RuneCountInString(returnToLobbyText))/2, displayHeight-6, boxStyle, returnToLobbyText)

	}

}

func (tg *TronGameView) renderGame(s *Screen) {
	tg.mgr.RLock()
	showDebug := tg.mgr.showDebug
	tg.mgr.RUnlock()
	for row := 0; row < tg.WorkingGameState.Width; row++ {
		for col := 0; col < tg.WorkingGameState.Height; col++ {
			if ok, playerNum := tg.getCollision(tg.WorkingGameState.Collisions, row, col); ok && playerNum >= 0 {
				style := tcell.StyleDefault.Background(tcell.ColorNames[TRON_COLORS[playerNum]])

				if showDebug {
					s.DrawText(row, col, style, "*")
				} else {
					s.DrawText(row, col, style, " ")
				}

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

// JANK: This applies entries in order without processing out of order timesteps. This could cause jumps in game state
// i.e. entries {timestep}: [A{32}, B{24}, C{28}]. This would be processed as [A{32}, B{33}, C{37}], but cmd C could be
// commited before timestep 37
// ^ maybe not applicable anymore
func (tg *TronGameView) startApplyChanHandler() {
	go func() {
		for {
			applyMsg := <-tg.ApplyChan
			// log.Println("[RAFT]", "APPLY")
			mu.Lock()
			tg.lastApplyMsgInd = applyMsg.CommandIndex - 1 // raft indexes are 1 indexed
			style := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)

			if applyMsg.CommandValid {
				if applyMsg.CommandTimestep < tg.CommitedGameState.CommitedTimeStep {
					panic(fmt.Sprintf("encountered older timestep than commitedTimestep, %d, %d", applyMsg.CommandTimestep, tg.CommitedGameState.CommitedTimeStep))
				} else if cmd, ok := readLogEntryAsTronCmd(applyMsg.Command); ok {
					log.Println("Applying: ", cmd, applyMsg.CommandTimestep)

					jumpAhead := math.Max(float64(applyMsg.CommandTimestep-tg.CommitedGameState.CommitedTimeStep-1), 0)
					log.Println("Jump ahead: ", jumpAhead)
					newCommitedGameState := tg.clientPredictAll(tg.CommitedGameState, int(jumpAhead))

					newCommitedGameState.CommitedTimeStep = applyMsg.CommandTimestep
					newCommitedGameState = tg.applyCommandToGameState(newCommitedGameState, cmd)
					newCommitedGameState = tg.clientPredictAll(newCommitedGameState, 1) // current timestep forward

					tg.CommitedGameState = newCommitedGameState

					tg.truncateMoveQueueIfNecessary(cmd)
				}

			}

			tg.mgr.RLock()
			if tg.mgr.showDebug {
				w, _ := tg.mgr.screen.displaySize()
				tg.mgr.screen.DrawText(w+1, 1, style, fmt.Sprintf("A:%d C: %d", tg.lastApplyMsgInd, tg.CommitedGameState.CommitedTimeStep))
			}
			tg.mgr.RUnlock()
			mu.Unlock()
			log.Println("Finished apply")
		}

	}()

}

func (tg *TronGameView) updateSelf() {
	// mu.Lock()
	// defer mu.Unlock()

	myState := tg.getMyState()

	if !myState.Alive {
		return
	}

	currentTimestep := tg.getTimestep()
	var cmd TronCommand

	if needToProcessInput {
		cmd = TronCommand{uuid.NewString(), TronMoveCmd, currentTimestep, tg.Me, tg.LatestInputDir, ""}
	} else if tg.NextDir != -1 {
		cmd = TronCommand{uuid.NewString(), TronMoveCmd, currentTimestep, tg.Me, tg.NextDir, ""}
		log.Println("use Nextdir")
		tg.NextDir = -1
	} else {
		return
	}

	tg.RaftServer.Start(cmd, currentTimestep)
	tg.MoveQueue = append(tg.MoveQueue, cmd)

	tg.mgr.RLock()
	if tg.mgr.showDebug {
		w, h := tg.mgr.screen.displaySize()
		moveStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorBlue)
		tg.mgr.screen.DrawText(w+1, h-1, moveStyle, cmd.String())
	}
	tg.mgr.RUnlock()

	// optimistically apply move
	myState.Direction = cmd.Direction
	tg.WorkingGameState.ClientStates[tg.Me] = myState

	needToProcessInput = false
}

func (tg *TronGameView) updateWorkingGameState(currentTimestep int) {

	// FUCK YOU RAFT WHY ARE YOU 1 INDEXED
	raftLog, lastApplied, _ := tg.RaftServer.GetLog()

	lastApplied -= 1

	allEntries := raftLog.GetEntries()
	partitionIndex := int(math.Min(float64(lastApplied)+1, float64(len(allEntries))))
	entries := allEntries[partitionIndex:]

	var commands BasicQueue[TronCommand]
	for _, entry := range entries {
		if cmd, ok := readLogEntryAsTronCmd(entry.Command); ok {

			cmd.Timestep = entry.Timestep
			commands.push(cmd)

			tg.truncateMoveQueueIfNecessary(cmd)
		}
	}

	// log.Println("working state", lastApplied, commitIndex, len(entries), len(tg.MoveQueue))

	// log.Println("cmds:", commands, "moveq", tg.MoveQueue)

	workingGameState := TronGameState{}
	copier.CopyWithOption(&workingGameState, &tg.CommitedGameState, copier.Option{DeepCopy: true})

	// JANK: mixing in move queue in to processed logs instead of replaying on top, could cause jumps
	// processedLogs.push(tg.MoveQueue...)
	if len(tg.MoveQueue) > 0 {
		if len(commands) > 0 && commands[len(commands)-1].Timestep > tg.MoveQueue[0].Timestep {
			diff := commands[len(commands)-1].Timestep - tg.MoveQueue[0].Timestep
			log.Println("[RAFT]", "diff", diff)
			for _, move := range tg.MoveQueue {
				move.Timestep += diff
				commands.push(move)
			}
		} else {
			commands.push(tg.MoveQueue...)
		}
	}

	// DEBUG MODE
	tg.mgr.RLock()
	if tg.mgr.showDebug {
		w, _ := tg.mgr.screen.displaySize()
		vOffset := 3
		var maxLogs float64 = 15

		tg.mgr.screen.DrawEmpty(w+1, vOffset, w+22, vOffset+len(allEntries)+3, tcell.StyleDefault.Background(tcell.ColorBlack))

		commitedStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen)
		lastI := 0
		startInd := int(math.Max(0, float64(partitionIndex)-maxLogs))
		for i, entry := range allEntries[startInd:partitionIndex] {
			if cmd, ok := readLogEntryAsTronCmd(entry.Command); ok {
				cmd.Timestep = entry.Timestep
				tg.mgr.screen.DrawText(w+1, vOffset+i, commitedStyle, cmd.String())
			}
			lastI = i
		}
		tg.mgr.screen.DrawText(w+1, vOffset+lastI+1, commitedStyle, fmt.Sprintf("appliedInd: %d", lastApplied))
		uncommitedStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
		for i, cmd := range commands {
			tg.mgr.screen.DrawText(w+1, vOffset+lastI+3+i, uncommitedStyle, cmd.String())
		}
	}
	tg.mgr.RUnlock()

	// currentTimestep := tg.getTimestep()
	// log.Println("CURENT TIMESTEP", currentTimestep)
	workingTimestep := workingGameState.CommitedTimeStep + 1

	// if len(commands) > 0 {
	// 	log.Println("AHH", commands[0].Timestep, workingTimestep, currentTimestep)
	// }

	// replay cmds on top of gamestate
	for len(commands) > 0 || workingTimestep <= currentTimestep {
		if len(commands) > 0 && commands[0].Timestep <= workingTimestep {
			if cmd, ok := commands.pop(); ok && cmd.Timestep == workingTimestep {
				workingGameState = tg.applyCommandToGameState(workingGameState, cmd)
			}
		} else {
			workingGameState = tg.clientPredictAll(workingGameState, 1)
			workingTimestep += 1
		}
	}

	if shouldWin, winner := tg.shouldWin(workingGameState); shouldWin {
		winCmd := TronCommand{uuid.NewString(), TronEndGameCmd, currentTimestep, tg.Me, -1, winner}
		tg.RaftServer.Start(winCmd, currentTimestep)
	}
	// fmt.Print("after: ", workingGameState.ClientStates)
	tg.WorkingGameState = workingGameState

}

// applies game state without increasing timestep
func (tg *TronGameView) applyCommandToGameState(gameState TronGameState, cmd TronCommand) TronGameState {
	clientState := gameState.ClientStates[cmd.PlayerID]
	switch cmd.Type {
	case TronMoveCmd:
		clientState.Direction = cmd.Direction
	case TronEndGameCmd:
		gameState.Ended = true
		gameState.Winner = cmd.Winner
	}
	gameState.ClientStates[cmd.PlayerID] = clientState
	return gameState
}

// blindly truncates move queue if id matches. Could potentially cut out earlier cmds in the moveQueue
func (tg *TronGameView) truncateMoveQueueIfNecessary(cmd TronCommand) {
	for i, move := range tg.MoveQueue {
		if move.Id == cmd.Id {
			tg.MoveQueue = tg.MoveQueue[i+1:]
			return
		}
	}
}

func (tg *TronGameView) clientPredictAll(gameState TronGameState, numTimesteps int) TronGameState {
	playerIds := make([]string, len(gameState.ClientStates))
	i := 0
	for k := range gameState.ClientStates {
		playerIds[i] = k
		i++
	}
	return tg.clientPredict(gameState, numTimesteps, playerIds)
}

func (tg *TronGameView) clientPredict(gameState TronGameState, numTimesteps int, playerIds []string) TronGameState {
	if gameState.Ended {
		return gameState
	}

	for i := 0; i < numTimesteps; i++ {
		for _, playerId := range playerIds {
			clientState := gameState.ClientStates[playerId]
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

// GAME FUNCTIONS
func (tg *TronGameView) getStartingPosAndDir() ([][2]int, []TronDirection) {
	width, height := tg.mgr.screen.displaySize()
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

func (tg *TronGameView) shouldWin(gameState TronGameState) (bool, string) {
	winner := ""
	if len(gameState.ClientStates) == 1 {
		winner = "can't win without friends :^)"
	}

	for id, client := range gameState.ClientStates {
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
	width, _ := tg.mgr.screen.displaySize()
	if !tg.isOutOfBounds(x, y) && playerNum < 8 {
		ind := y*width + x
		collisions[ind/2] |= byte(playerNum<<1+1) << ((ind % 2) * 4)
	}
	return collisions
}

func (tg *TronGameView) getTimestep() int {
	return tg.RaftServer.GetTimestep()
}

// returns bool of collision and player num
func (tg *TronGameView) getCollision(collisions []byte, x int, y int) (bool, int) {
	width, _ := tg.mgr.screen.displaySize()
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

func canMoveInDir(currentDir TronDirection, proposedDir TronDirection) bool {
	if currentDir == TronDown || currentDir == TronUp {
		return proposedDir == TronLeft || proposedDir == TronRight
	}
	if currentDir == TronLeft || currentDir == TronRight {
		return proposedDir == TronDown || proposedDir == TronUp
	}
	return false
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

func (tg *TronGameView) toByteStr(collisions [][]bool) string {
	width, height := tg.mgr.screen.displaySize()
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

func (tg *TronGameView) fromBytestr(byteStr string) [][]bool {
	width, height := tg.mgr.screen.displaySize()
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

func (tg *TronGameView) initCollisions() []byte {
	width, height := tg.mgr.screen.displaySize()
	return make([]byte, int(math.Ceil(float64(width*height)/2)))
}

func (v *TronGameView) GetHeartbeatMetadata() encoding.BinaryMarshaler {
	return nil
}

func (tg *TronGameView) Unload() {
	tg.RaftServer.Kill()
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
