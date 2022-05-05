package arcade

import (
	"encoding"
	"math"
	"time"

	"github.com/gdamore/tcell/v2"
)

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

type TronGameState struct {
	Width  int
	Height int
	Ended  bool
	// Collisions [][]bool
}

var Collisions [][]bool

type TronClientState struct {
	Timestep  int
	Alive     bool
	Color     string
	PathX     []int
	PathY     []int
	X         int
	Y         int
	Direction TronDirection
}

type TronGameView struct {
	View
	Game[TronGameState, TronClientState]
}

const CLIENT_LAG_TIMESTEP = 2

func NewTronGameView(lobby *Lobby) *TronGameView {
	return &TronGameView{
		Game: Game[TronGameState, TronClientState]{
			ID:             lobby.ID,
			PlayerIDs:      lobby.PlayerIDs,
			Name:           lobby.Name,
			Me:             arcade.Server.ID,
			HostID:         lobby.HostID,
			HostSyncPeriod: 1000,
			TimestepPeriod: 200,
			Timestep:       0,
		},
	}
}

var lastReceivedInp = make(map[string]int)

func (tg *TronGameView) Init() {
	width, height := arcade.ViewManager.screen.displaySize()
	collisions := make([][]bool, width)
	for i := range collisions {
		collisions[i] = make([]bool, height)
	}
	Collisions = collisions
	tg.GameState = TronGameState{width, height, false} //, collisions}

	clientState := make(map[string]TronClientState)
	startingPos, startingDir := getStartingPosAndDir()
	for i, playerID := range tg.PlayerIDs {
		x := startingPos[i][0]
		y := startingPos[i][1]
		clientState[playerID] = TronClientState{tg.Timestep, true, TRON_COLORS[i], []int{x}, []int{y}, x, y, startingDir[i]}
		lastReceivedInp[playerID] = 0
	}

	tg.ClientStates = clientState

	tg.start()

	go func() {
		for !tg.GameState.Ended {
			tg.Timestep += 1
			tg.updateSelf()
			arcade.ViewManager.RequestRender()
			time.Sleep(time.Duration(tg.TimestepPeriod * int(time.Millisecond)))
			tg.updateOthers()
		}
	}()
}

func (tg *TronGameView) ProcessEvent(ev interface{}) {
	switch ev := ev.(type) {
	case *ClientDisconnectEvent:
		// process disconnected client
	case *tcell.EventKey:
		tg.ProcessEventKey(ev)
	}
}

func (tg *TronGameView) ProcessEventKey(ev *tcell.EventKey) {
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
	}
	tg.setMyState(state)
}

func (tg *TronGameView) ProcessMessage(from *Client, p interface{}) interface{} {
	switch p := p.(type) {
	case GameUpdateMessage[TronGameState, TronClientState]:
		// tg.handleGameUpdate(p)
	case ClientUpdateMessage[TronClientState]:
		tg.handleClientUpdate(p)
	}
	return nil
}

func (tg *TronGameView) Render(s *Screen) {

	// defaultStyle := tcell.StyleDefault.Background(tcell.ColorTeal).Foreground(tcell.ColorTeal)
	// s.DrawLine(0, 0, tg.GameState.width, 0, style, true)
	// s.DrawLine(tg.GameState.width, 0, tg.GameState.width, tg.GameState.height, style, true)
	// s.DrawLine(0, tg.GameState.height, tg.GameState.width, tg.GameState.height, style, true)
	// s.DrawLine(0, 0, 0, tg.GameState.height, style, true)
	s.ClearContent()

	// displayWidth, displayHeight := s.displaySize()
	// s.DrawBox(0, 0, displayWidth-1, displayHeight-1, defaultStyle, true)

	for _, client := range tg.ClientStates {
		style := tcell.StyleDefault.Background(tcell.ColorNames[client.Color])
		hStyle := tcell.StyleDefault.Foreground(tcell.ColorNames[client.Color]) //.Foreground(tcell.ColorWhite)
		// vStyle := tcell.StyleDefault.Background(tcell.ColorNames[client.Color])
		for i := 0; i < len(client.PathX)-1; i++ {
			// s.DrawText(client.PathX[i], client.PathY[i], style, " ")
			// if math.Abs(float64(client.PathX[i+1]-client.PathX[i])) > 0 {
			// 	s.DrawText(client.PathX[i], client.PathY[i], hStyle, "■")
			// } else if math.Abs(float64(client.PathY[i+1]-client.PathY[i])) > 0 {
			// 	s.DrawText(client.PathX[i], client.PathY[i], vStyle, " ")
			// }

			if math.Abs(float64(client.PathX[i+1]-client.PathX[i])) > 0 && i < len(client.PathX)-2 {
				s.DrawText(client.PathX[i], client.PathY[i], style, " ")
			} else if math.Abs(float64(client.PathY[i+1]-client.PathY[i])) > 0 {
				s.DrawText(client.PathX[i], client.PathY[i], style, " ")
			}

		}

		if client.Alive {
			style := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorNames[client.Color])
			chr := getDirChr(client.Direction)
			s.DrawText(client.X, client.Y, style, chr)
		} else {
			s.DrawText(client.X, client.Y, hStyle, "😵")
		}

	}
}

func (tg *TronGameView) updateState() {

}

func (tg *TronGameView) updateSelf() {

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

	tg.setCollision(state.X, state.Y)
	state.PathX = append(state.PathX, state.X)
	state.PathY = append(state.PathY, state.Y)

	state.Timestep = tg.Timestep
	tg.setMyState(state)
	tg.sendClientUpdate(state)
}

func (tg *TronGameView) updateOthers() {
	for id, state := range tg.ClientStates {
		if id != tg.Me && lastReceivedInp[id]+CLIENT_LAG_TIMESTEP < tg.Timestep {
			tg.ClientStates[id] = tg.clientPredict(state, tg.Timestep)
		}
	}
}

func (tg *TronGameView) clientPredict(state TronClientState, targetTimestep int) TronClientState {
	if !state.Alive || targetTimestep <= state.Timestep {
		return state
	}
	delta := targetTimestep - state.Timestep

	newPathX := []int{}
	newPathY := []int{}

	lastX := state.PathX[len(state.PathX)-1]
	lastY := state.PathY[len(state.PathY)-1]
	// for i := 1; i <= delta; i++ {
	// 	switch state.Direction {
	// 	case TronUp:
	// 		newPathX = append(newPathX, lastX)
	// 		newPathY = append(newPathY, lastY-i)
	// 	case TronRight:
	// 		newPathX = append(newPathX, lastX+i)
	// 		newPathY = append(newPathY, lastY)
	// 	case TronDown:
	// 		newPathX = append(newPathX, lastX)
	// 		newPathY = append(newPathY, lastY+i)
	// 	case TronLeft:
	// 		newPathX = append(newPathX, lastX-i)
	// 		newPathY = append(newPathY, lastY)
	// 	}
	// }

	for i := 0; i < delta-1; i++ {
		newPathX = append(newPathX, lastX)
		newPathY = append(newPathY, lastY)
	}

	switch state.Direction {
	case TronUp:
		newPathX = append(newPathX, lastX)
		newPathY = append(newPathY, lastY-1)
	case TronRight:
		newPathX = append(newPathX, lastX+1)
		newPathY = append(newPathY, lastY)
	case TronDown:
		newPathX = append(newPathX, lastX)
		newPathY = append(newPathY, lastY+1)
	case TronLeft:
		newPathX = append(newPathX, lastX-1)
		newPathY = append(newPathY, lastY)
	}

	state.Timestep = targetTimestep
	state.PathX = append(state.PathX, newPathX...)
	state.PathY = append(state.PathY, newPathY...)
	state.X = newPathX[len(newPathX)-1]
	state.Y = newPathY[len(newPathY)-1]

	if tg.shouldDie(state) {
		state = tg.die(state)
	}
	tg.setCollision(state.X, state.Y)

	state.Timestep = tg.Timestep
	return state
}

func (tg *TronGameView) handleGameUpdate(data GameUpdateMessage[TronGameState, TronClientState]) {
	tg.GameState = data.GameUpdate
	for id, clientState := range data.ClientStates {
		if id != tg.Me {
			tg.ClientStates[id] = clientState
		}
	}
	tg.recalculateCollisions()
}

func (tg *TronGameView) handleClientUpdate(data ClientUpdateMessage[TronClientState]) {

	update := data.Update
	tg.ClientStates[data.Id] = update
	arcade.ViewManager.RequestRender()
	tg.recalculateCollisions()

	lastReceivedInp[data.Id] = update.Timestep
}

func (tg *TronGameView) recalculateCollisions() {
	collisions := make([][]bool, tg.GameState.Width)
	for i := range collisions {
		collisions[i] = make([]bool, tg.GameState.Height)
	}
	Collisions = collisions
	for _, player := range tg.ClientStates {
		for i := 0; i < len(player.PathX); i++ {
			tg.setCollision(player.PathX[i], player.PathY[i])
		}
	}
}

// GAME FUNCTIONS

func getStartingPosAndDir() ([][2]int, []TronDirection) {
	width, height := arcade.ViewManager.screen.displaySize()
	margin := int(math.Round(math.Min(float64(width)/10, float64(height)/10)))
	return [][2]int{{margin, margin}, {width - margin, height - margin}, {width - margin, margin}, {margin, height - margin}, {width / 2, margin}, {width - margin, height / 2}, {width / 2, height - margin}, {margin, height / 2}}, []TronDirection{TronRight, TronLeft, TronDown, TronUp, TronDown, TronLeft, TronUp, TronRight}
}

func (tg *TronGameView) shouldDie(player TronClientState) bool {
	return tg.isOutOfBounds(player.X, player.Y) || Collisions[player.X][player.Y]
}

func (tg *TronGameView) die(player TronClientState) TronClientState {
	player.Alive = false
	return player
}

func (tg *TronGameView) isOutOfBounds(x int, y int) bool {
	return x <= 0 || x >= tg.GameState.Width-1 || y <= 0 || y >= tg.GameState.Height-1
}

func (tg *TronGameView) setCollision(x int, y int) {
	if !tg.isOutOfBounds(x, y) {
		Collisions[x][y] = true
	}
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

func (v *TronGameView) GetHeartbeatMetadata() encoding.BinaryMarshaler {
	return nil
}
