package arcade

import (
	"encoding"
	"fmt"
	"math"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/google/uuid"
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

type TronGameState struct {
	Width      int
	Height     int
	Ended      bool
	Winner     string
	Collisions []byte
}

// use this to coalesce optimistic game state
// in the future, generalize this
var localCollisions []byte

type TronClientState struct {
	Timestep       int
	Alive          bool
	Color          string
	PathX          []int
	PathY          []int
	X              int
	Y              int
	Direction      TronDirection
	CommitTimestep int
	PlayerNum      int
}

type TronGameView struct {
	View
	Game[TronGameState, TronClientState]
}

const CLIENT_LAG_TIMESTEP = 0
const FRAGMENTS = 2

func NewTronGameView(lobby *Lobby) *TronGameView {
	return &TronGameView{
		Game: Game[TronGameState, TronClientState]{
			ID:             lobby.ID,
			PlayerIDs:      lobby.PlayerIDs,
			Name:           lobby.Name,
			Me:             arcade.Server.ID,
			HostID:         lobby.HostID,
			HostSyncPeriod: 2000,
			TimestepPeriod: 200,
			Timestep:       0,
		},
	}
}

var lastReceivedInp = make(map[string]int)
var processedInp = false

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

func (tg *TronGameView) Init() {
	width, height := arcade.ViewManager.screen.displaySize()
	tg.GameState = TronGameState{width, height, false, "", initCollisions()}

	localCollisions = initCollisions()

	clientState := make(map[string]TronClientState)
	startingPos, startingDir := getStartingPosAndDir()
	for i, playerID := range tg.PlayerIDs {
		x := startingPos[i][0]
		y := startingPos[i][1]
		clientState[playerID] = TronClientState{tg.Timestep, true, TRON_COLORS[i], []int{x}, []int{y}, x, y, startingDir[i], 0, i}
		lastReceivedInp[playerID] = 0
	}

	tg.ClientStates = clientState

	tg.start()

	go func() {
		for i := 3; i > 0; i-- {
			countdownNum = i
			arcade.ViewManager.RequestRender()
			time.Sleep(time.Duration(int(time.Second)))
		}

		gameRenderState = TronGameScreen
		for !tg.GameState.Ended {
			tg.Timestep += 1
			processedInp = false
			tg.updateSelf()
			arcade.ViewManager.RequestRender()
			time.Sleep(time.Duration(tg.TimestepPeriod * int(time.Millisecond)))
			tg.updateOthers()

			mu.Lock()
			if tg.Me == tg.HostID {
				if ended, winner := tg.shouldWin(); ended {
					tg.GameState.Ended = ended
					tg.GameState.Winner = winner
					tg.sendGameUpdate()
					tg.sendEndGame(winner)
				}
			}
			mu.Unlock()
		}

		gameRenderState = TronWinScreen
		arcade.ViewManager.RequestRender()
	}()

	if tg.Me == tg.HostID && tg.HostSyncPeriod > 0 {
		go tg.startHostSync()
	}

}

func (tg *TronGameView) startHostSync() {
	for tg.Started {
		time.Sleep(time.Duration(tg.HostSyncPeriod * int(time.Millisecond)))
		tg.commitGameState()
		tg.sendGameUpdate()
	}
}

func (tg *TronGameView) ProcessEvent(ev interface{}) {
	switch ev := ev.(type) {
	case *ClientDisconnectEvent:
		// process disconnected client
	case *tcell.EventKey:
		if ev.Key() == tcell.KeyEnter {
			mu.Lock()
			gamestate := tg.GameState.Ended
			mu.Unlock()
			if gamestate {
				// arcade.Lobby.mu.RLock()
				// hostID := arcade.Lobby.HostID
				// lobbyID := arcade.Lobby.ID
				// arcade.Lobby.mu.RUnlock()

				// if arcade.Server.ID == hostID {
				arcade.lobbyMux.Lock()
				arcade.Lobby = &Lobby{}
				arcade.lobbyMux.Unlock()

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
	if processedInp {
		return
	}
	key := ev.Key()
	state := tg.getMyState()
	switch key {
	case tcell.KeyUp:
		if state.Direction != TronDown {
			state.Direction = TronUp
		}
	case tcell.KeyRight:
		if state.Direction != TronLeft {
			state.Direction = TronRight
		}
	case tcell.KeyDown:
		if state.Direction != TronUp {
			state.Direction = TronDown
		}
	case tcell.KeyLeft:
		if state.Direction != TronRight {
			state.Direction = TronLeft
		}
	case tcell.KeyCtrlG:
		showCommits = !showCommits

	}
	tg.setMyState(state)
	processedInp = true
}

func (tg *TronGameView) ProcessMessage(from *Client, p interface{}) interface{} {
	switch p := p.(type) {
	case GameUpdateMessage[TronGameState, TronClientState]:
		tg.handleGameUpdate(p)
	case ClientUpdateMessage[TronClientState]:
		tg.handleClientUpdate(p)
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
		if tg.GameState.Winner == tg.Me {
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

	for row := 0; row < tg.GameState.Width; row++ {
		for col := 0; col < tg.GameState.Height; col++ {
			if ok, playerNum := tg.getCollision(localCollisions, row, col); ok && playerNum >= 0 {
				style := tcell.StyleDefault.Background(tcell.ColorNames[TRON_COLORS[playerNum]])
				s.DrawText(row, col, style, " ")
			}

			if showCommits {
				if ok, playerNum := tg.getCollision(tg.GameState.Collisions, row, col); ok && playerNum >= 0 && playerNum < len(TRON_COLORS)-1 {
					style := tcell.StyleDefault.Background(tcell.ColorNames[TRON_COLORS[playerNum+1]])
					s.DrawText(row, col, style, " ")
				}
			}
		}
	}

	for _, client := range tg.ClientStates {
		style := tcell.StyleDefault.Background(tcell.ColorNames[client.Color])

		for i := 0; i < len(client.PathX)-1; i++ {
			s.DrawText(client.PathX[i], client.PathY[i], style, " ")
		}

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
	state := tg.getMyState()
	if !state.Alive {
		return
	}
	switch state.Direction {
	case TronUp:
		state.Y -= 1
	case TronRight:
		state.X += 1
	case TronDown:
		state.Y += 1
	case TronLeft:
		state.X -= 1
	}

	if tg.shouldDie(state) {
		state = tg.die(state)
	}

	localCollisions = tg.setCollision(localCollisions, state.X, state.Y, state.PlayerNum)
	state.PathX = append(state.PathX, state.X)
	state.PathY = append(state.PathY, state.Y)

	state.Timestep = tg.Timestep
	lastReceivedInp[tg.Me] = tg.Timestep

	tg.setMyState(state)
	tg.sendClientUpdate(state)

}

// lock this shit
func (tg *TronGameView) updateOthers() {
	mu.Lock()
	for id, state := range tg.ClientStates {
		if id != tg.Me && lastReceivedInp[id]+CLIENT_LAG_TIMESTEP < tg.Timestep {

			tg.ClientStates[id] = tg.clientPredict(state, tg.Timestep)

		}
	}
	mu.Unlock()
}

func (tg *TronGameView) clientPredict(state TronClientState, targetTimestep int) TronClientState {
	if !state.Alive || targetTimestep <= state.Timestep {
		return state
	}
	delta := targetTimestep - state.Timestep

	newPathX := []int{}
	newPathY := []int{}

	lastX := state.X
	lastY := state.Y
	for i := 1; i <= delta; i++ {
		switch state.Direction {
		case TronUp:
			newPathX = append(newPathX, lastX)
			newPathY = append(newPathY, lastY-i)
		case TronRight:
			newPathX = append(newPathX, lastX+i)
			newPathY = append(newPathY, lastY)
		case TronDown:
			newPathX = append(newPathX, lastX)
			newPathY = append(newPathY, lastY+i)
		case TronLeft:
			newPathX = append(newPathX, lastX-i)
			newPathY = append(newPathY, lastY)
		}
	}

	state.Timestep = targetTimestep
	state.PathX = append(state.PathX, newPathX...)
	state.PathY = append(state.PathY, newPathY...)
	state.X = newPathX[len(newPathX)-1]
	state.Y = newPathY[len(newPathY)-1]

	if tg.shouldDie(state) {
		state = tg.die(state)
	}

	localCollisions = tg.setCollision(localCollisions, state.X, state.Y, state.PlayerNum)

	state.Timestep = tg.Timestep
	return state
}

func (tg *TronGameView) handleGameUpdate(data GameUpdateMessage[TronGameState, TronClientState]) {
	mu.Lock()
	defer mu.Unlock()
	if data.ID != currGameUpdateId {
		currGameUpdateId = data.ID
		currFragments = 0
		currCollisions = initCollisions()
	}

	gameState := data.GameUpdate
	size := len(currCollisions)
	currCollisions = append(append(currCollisions[:data.FragmentNum*size/FRAGMENTS], gameState.Collisions...), currCollisions[int(math.Min(float64(size), float64((data.FragmentNum+1)*size/FRAGMENTS))):]...)
	currFragments += 1

	if currFragments != FRAGMENTS {
		return
	}
	gameState.Collisions = currCollisions
	tg.GameState = gameState

	for id, lastInp := range data.LastInps {
		currClient := tg.ClientStates[id]
		if lastInp >= currClient.CommitTimestep {
			diff := lastInp - currClient.CommitTimestep

			currClient.CommitTimestep = lastInp
			currClient.PathX = currClient.PathX[int(math.Min(float64(diff), float64(len(currClient.PathX)))):]
			currClient.PathY = currClient.PathY[int(math.Min(float64(diff), float64(len(currClient.PathY)))):]
			if lastReceivedInp[id] < lastInp {
				lastReceivedInp[id] = lastInp
			}
			tg.ClientStates[id] = currClient
		} else {
			panic("incoming client < currclient commitTimestep")
		}
	}
	tg.recalculateCollisions()
}

func (tg *TronGameView) handleClientUpdate(data ClientUpdateMessage[TronClientState]) {
	mu.Lock()
	defer mu.Unlock()
	update := data.Update
	state := tg.ClientStates[data.Id]

	if state.CommitTimestep <= update.CommitTimestep {
		if update.CommitTimestep <= state.Timestep+1 {
			diff := update.CommitTimestep - state.CommitTimestep
			update.PathX = append(state.PathX[:diff], update.PathX...)
			update.PathY = append(state.PathY[:diff], update.PathY...)
		} else {
			update = tg.clientPredict(state, update.CommitTimestep-1) // client predict to fill gap
			update.PathX = append(state.PathX, update.PathX...)
			update.PathY = append(state.PathY, update.PathY...)
		}
	} else {
		if update.Timestep >= state.CommitTimestep {
			diff := state.CommitTimestep - update.CommitTimestep
			update.PathX = update.PathX[diff:]
			update.PathY = update.PathY[diff:]
		} else {
			panic("update out of date: " + fmt.Sprintf("%d [%d] < %d [%d]", update.CommitTimestep, len(update.PathX), state.CommitTimestep, len(state.PathX)))
		}
	}

	update.CommitTimestep = state.CommitTimestep

	tg.ClientStates[data.Id] = update
	arcade.ViewManager.RequestRender()
	tg.recalculateCollisions()

	lastReceivedInp[data.Id] = update.Timestep
}

func (tg *TronGameView) handleEndGame(data EndGameMessage) {
	tg.GameState.Ended = true
	tg.GameState.Winner = data.Winner
	arcade.ViewManager.RequestRender()
}

func (tg *TronGameView) commitGameState() {
	mu.Lock()
	defer mu.Unlock()
	collisions := tg.GameState.Collisions
	for id, player := range tg.ClientStates {
		for i := 0; i < lastReceivedInp[id]-player.CommitTimestep; i++ {
			collisions = tg.setCollision(collisions, player.PathX[i], player.PathY[i], player.PlayerNum)
		}

		diff := lastReceivedInp[id] - player.CommitTimestep
		player.CommitTimestep = lastReceivedInp[id]
		player.PathX = player.PathX[int(math.Min(float64(diff), float64(len(player.PathX)))):]
		player.PathY = player.PathY[int(math.Min(float64(diff), float64(len(player.PathY)))):]

		tg.ClientStates[id] = player
	}
	tg.GameState.Collisions = collisions
	tg.recalculateCollisions()
}

func (g *Game[GS, CS]) sendEndGame(winner string) {
	endGame := &EndGameMessage{Message: Message{Type: "end_game"}, Winner: winner}

	for clientId := range g.ClientStates {
		if client, ok := arcade.Server.Network.GetClient(clientId); ok && clientId != g.Me {
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
	clientUpdate := &ClientUpdateMessage[CS]{Message: Message{Type: "client_update"}, Id: g.Me, Update: update}

	for clientId := range g.ClientStates {
		if client, ok := arcade.Server.Network.GetClient(clientId); ok && clientId != g.Me {
			arcade.Server.Network.Send(client, clientUpdate)
		}
	}
}

func (tg *TronGameView) sendGameUpdate() {
	if tg.Me != tg.HostID {
		return
	}

	arcade.Server.RLock()
	defer arcade.Server.RUnlock()

	for clientId := range tg.ClientStates {
		if client, ok := arcade.Server.Network.GetClient(clientId); ok && clientId != tg.Me {
			id := uuid.NewString()
			for i := 0; i < FRAGMENTS; i++ {
				gameState := tg.GameState
				size := len(gameState.Collisions)
				frag := gameState.Collisions[i*size/FRAGMENTS : int(math.Min(float64(size), float64((i+1)*size/FRAGMENTS)))]
				gameState.Collisions = frag
				data := &GameUpdateMessage[TronGameState, TronClientState]{Message{Type: "game_update"}, gameState, lastReceivedInp, id, i}
				arcade.Server.Network.Send(client, data)
			}

		}
	}
}

func (tg *TronGameView) recalculateCollisions() {
	copy(localCollisions, tg.GameState.Collisions)
	for _, player := range tg.ClientStates {
		for i := 0; i < len(player.PathX); i++ {
			localCollisions = tg.setCollision(localCollisions, player.PathX[i], player.PathY[i], player.PlayerNum)
		}
	}
}

// GAME FUNCTIONS
func getStartingPosAndDir() ([][2]int, []TronDirection) {
	width, height := arcade.ViewManager.screen.displaySize()
	width -= 1 // account for tron border
	height -= 1
	margin := int(math.Round(math.Min(float64(width)/8, float64(height)/8)))
	return [][2]int{{margin, margin}, {width - margin, height - margin}, {width - margin, margin}, {margin, height - margin}, {width / 2, margin}, {width - margin, height / 2}, {width / 2, height - margin}, {margin, height / 2}}, []TronDirection{TronRight, TronLeft, TronDown, TronUp, TronDown, TronLeft, TronUp, TronRight}
}

func (tg *TronGameView) shouldDie(player TronClientState) bool {
	collides, _ := tg.getCollision(localCollisions, player.X, player.Y)
	return tg.isOutOfBounds(player.X, player.Y) || collides
}

func (tg *TronGameView) die(player TronClientState) TronClientState {
	player.Alive = false
	return player
}

func (tg *TronGameView) shouldWin() (bool, string) {
	winner := ""
	if len(tg.ClientStates) == 1 {
		winner = "can't win without friends :^)"
	}

	for id, client := range tg.ClientStates {
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
	return x <= 1 || x >= tg.GameState.Width-2 || y <= 1 || y >= tg.GameState.Height-2
}

func (tg *TronGameView) setCollision(collisions []byte, x int, y int, playerNum int) []byte {
	width, _ := arcade.ViewManager.screen.displaySize()
	if !tg.isOutOfBounds(x, y) && playerNum < 8 {
		ind := y*width + x
		collisions[ind/2] |= byte(playerNum<<1+1) << ((ind % 2) * 4)
	}
	return collisions
}

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
	return tg.ClientStates[tg.Me]
}

func (tg *TronGameView) setMyState(state TronClientState) {
	tg.ClientStates[tg.Me] = state
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
