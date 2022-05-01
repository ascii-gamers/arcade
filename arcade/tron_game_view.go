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
	timestep  int
	alive     bool
	color     string
	pathX     []int
	pathY     []int
	x         int
	y         int
	direction TronDirection
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
			TimestepPeriod: 1000,
			Timestep:       0,
		},
	}
}

func (tg *TronGameView) Init() {
	width, height := mgr.screen.displaySize()
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
		for {
			tg.Timestep += 1
			tg.updateSelf()
			time.Sleep(time.Duration(tg.TimestepPeriod * int(time.Millisecond)))
			tg.updateOthers()

			arcade.ViewManager.RequestRender()
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
		state.direction = TronUp
	case tcell.KeyRight:
		state.direction = TronRight
	case tcell.KeyDown:
		state.direction = TronDown
	case tcell.KeyLeft:
		state.direction = TronLeft
	}
	tg.setMyState(state)
}

func (tg *TronGameView) ProcessMessage(from *Client, p interface{}) interface{} {
	switch p := p.(type) {
	case GameUpdateMessage[TronGameState, TronClientState]:
		tg.handleGameUpdate(p)
	case ClientUpdateMessage[TronClientState]:
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

	for _, client := range tg.ClientStates {
		for i := 0; i < len(client.pathX); i++ {
			s.DrawText(client.pathX[i], client.pathY[i], style, "*")
		}
		if client.alive {
			s.DrawText(client.x, client.y, style, "😎")
		} else {
			s.DrawText(client.x, client.y, style, "😵")
		}

	}
}

func (tg *TronGameView) updateSelf() {
	// tg.state.direction = (tg.ClientStates[tg.Me].direction + 1 % 4)

	state := tg.getMyState()
	if !state.alive {
		return
	}
	switch state.direction {
	case TronUp:
		state.y -= 1
	case TronRight:
		state.x += 1
	case TronDown:
		state.y += 1
	case TronLeft:
		state.x -= 1
	}
	// fmt.Println(tg.state, tg.GameState.collisions[36])
	if tg.shouldDie(state) {
		state = tg.die(state)
	}

	tg.setCollision(state.x, state.y)
	state.pathX = append(state.pathX, state.x)
	state.pathY = append(state.pathY, state.y)

	state.timestep = tg.Timestep
	tg.setMyState(state)
	tg.sendClientUpdate(state)
}

func (tg *TronGameView) updateOthers() {
	for id, state := range tg.ClientStates {
		if id != tg.Me && state.timestep < tg.Timestep {
			tg.ClientStates[id] = tg.clientPredict(state, tg.Timestep)
		}
	}
}

func (tg *TronGameView) clientPredict(state TronClientState, targetTimestep int) TronClientState {

	if !state.alive || targetTimestep <= state.timestep {
		return state
	}
	delta := targetTimestep - state.timestep

	newPathX := []int{}
	newPathY := []int{}

	lastX := state.pathX[len(state.pathX)-1]
	lastY := state.pathY[len(state.pathY)-1]
	for i := 1; i <= delta; i++ {
		switch state.direction {
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
	state.timestep = targetTimestep
	state.pathX = append(state.pathX, newPathX...)
	state.pathY = append(state.pathY, newPathY...)
	state.x = newPathX[len(newPathX)-1]
	state.y = newPathY[len(newPathY)-1]

	if tg.shouldDie(state) {
		state = tg.die(state)
	}
	tg.setCollision(state.x, state.y)

	state.timestep = tg.Timestep
	return state
}

func (tg *TronGameView) handleGameUpdate(data GameUpdateMessage[TronGameState, TronClientState]) {
	tg.GameState = data.GameUpdate
	tg.ClientStates = data.ClientStates

}

func (tg *TronGameView) handleClientUpdate(data ClientUpdateMessage[TronClientState]) {
	tg.ClientStates[data.Id] = data.Update
}

// GAME FUNCTIONS

func (tg *TronGameView) shouldDie(player TronClientState) bool {

	// fmt.Println(tg.isOutOfBounds(player.x, player.y), player.x, player.y, tg.GameState.collisions[player.x][player.y])
	return tg.isOutOfBounds(player.x, player.y) || tg.GameState.collisions[player.x][player.y]
}

func (tg *TronGameView) die(player TronClientState) TronClientState {
	player.alive = false
	return player
}

func (tg *TronGameView) isOutOfBounds(x int, y int) bool {
	return x <= 0 || x >= tg.GameState.width || y <= 0 || y >= tg.GameState.height
}

func (tg *TronGameView) setCollision(x int, y int) {
	// fmt.Println("col: ", x, y)
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
