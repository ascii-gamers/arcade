package arcade

import (
	"math/rand"
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

type TronGameState struct {
	width      int
	height     int
	ended      bool
	collisions [][]bool
}

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

func NewTronGameView(lobby *Lobby) *TronGameView {
	return &TronGameView{
		Game: Game[TronGameState, TronClientState]{
			ID:             lobby.ID,
			PlayerIDs:      lobby.PlayerIDs,
			Name:           lobby.Name,
			Me:             arcade.Server.ID,
			HostID:         lobby.HostID,
			HostSyncPeriod: 1000,
			TimestepPeriod: 300,
			Timestep:       0,
		},
	}
}

func (tg *TronGameView) Init() {
	width, height := arcade.ViewManager.screen.displaySize()
	collisions := make([][]bool, width)
	for i := range collisions {
		collisions[i] = make([]bool, height)
	}
	tg.GameState = TronGameState{width, height, false, collisions}

	clientState := make(map[string]TronClientState)
	for i, playerID := range tg.PlayerIDs {
		x := width/2 + rand.Intn(10) - 5
		y := height/2 + rand.Intn(10) - 5
		clientState[playerID] = TronClientState{tg.Timestep, true, TRON_COLORS[i], []int{x}, []int{y}, x, y, TronDown}
	}

	tg.ClientStates = clientState
	// tg.state = clientState[tg.Me]

	tg.start()

	go func() {
		for !tg.GameState.ended {
			tg.Timestep += 1
			tg.updateSelf()
			arcade.ViewManager.RequestRender()
			time.Sleep(time.Duration(tg.TimestepPeriod * int(time.Millisecond)))
			tg.updateOthers()

			// arcade.ViewManager.RequestRender()
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
		state.Direction = TronUp
	case tcell.KeyRight:
		state.Direction = TronRight
	case tcell.KeyDown:
		state.Direction = TronDown
	case tcell.KeyLeft:
		state.Direction = TronLeft
	}
	tg.setMyState(state)
}

func (tg *TronGameView) ProcessMessage(from *Client, p interface{}) interface{} {
	switch p := p.(type) {
	case GameUpdateMessage[TronGameState, TronClientState]:
		// tg.handleGameUpdate(p)
	case ClientUpdateMessage:
		tg.handleClientUpdate(p)
	}
	return nil
}

func (tg *TronGameView) Render(s *Screen) {
	style := tcell.StyleDefault.Background(tcell.ColorDarkGreen).Foreground(tcell.ColorWhite)

	// s.DrawLine(0, 0, tg.GameState.width, 0, style, true)
	// s.DrawLine(tg.GameState.width, 0, tg.GameState.width, tg.GameState.height, style, true)
	// s.DrawLine(0, tg.GameState.height, tg.GameState.width, tg.GameState.height, style, true)
	// s.DrawLine(0, 0, 0, tg.GameState.height, style, true)
	s.ClearContent()
	for _, client := range tg.ClientStates {
		for i := 0; i < len(client.PathX)-1; i++ {
			s.DrawText(client.PathX[i], client.PathY[i], style, "*")
		}
		if client.Alive {
			s.DrawText(client.X, client.Y, style, "😎")
		} else {
			s.DrawText(client.X, client.Y, style, "😵")
		}

	}
}

func (tg *TronGameView) updateState() {

}

func (tg *TronGameView) updateSelf() {
	// tg.state.direction = (tg.ClientStates[tg.Me].direction + 1 % 4)

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
	// fmt.Println(tg.state, tg.GameState.collisions[36])
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
		if id != tg.Me && state.Timestep < tg.Timestep {
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
	tg.setCollision(state.X, state.Y)

	state.Timestep = tg.Timestep
	return state
}

func (tg *TronGameView) handleGameUpdate(data GameUpdateMessage[TronGameState, TronClientState]) {
	tg.GameState = data.GameUpdate
	tg.ClientStates = data.ClientStates

}

func (tg *TronGameView) handleClientUpdate(data ClientUpdateMessage) {
	state := data.TronClientState
	tg.setCollision(state.X, state.Y)
	tg.ClientStates[data.Id] = state
	arcade.ViewManager.RequestRender()
}

// GAME FUNCTIONS

func (tg *TronGameView) shouldDie(player TronClientState) bool {

	// fmt.Println(tg.isOutOfBounds(player.x, player.y), player.x, player.y, tg.GameState.collisions[player.x][player.y])
	return tg.isOutOfBounds(player.X, player.Y) || tg.GameState.collisions[player.X][player.Y]
}

func (tg *TronGameView) die(player TronClientState) TronClientState {
	player.Alive = false
	return player
}

func (tg *TronGameView) isOutOfBounds(x int, y int) bool {
	return x <= 0 || x >= tg.GameState.width || y <= 0 || y >= tg.GameState.height
}

func (tg *TronGameView) setCollision(x int, y int) {
	if !tg.isOutOfBounds(x, y) {
		tg.GameState.collisions[x][y] = true
	}
}

func (tg *TronGameView) getMyState() TronClientState {
	return tg.ClientStates[tg.Me]
}

func (tg *TronGameView) setMyState(state TronClientState) {
	tg.ClientStates[tg.Me] = state
}
