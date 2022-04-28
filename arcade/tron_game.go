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

type TronGame struct {
	View

	Game[TronGameState, TronClientState]
	state    TronClientState
	renderCh chan int
}

func NewTronGame(pendingGame *PendingGame) *TronGame {
	return &TronGame{Game: Game[TronGameState, TronClientState]{PlayerList: pendingGame.PlayerList, Name: pendingGame.Name, Me: server.ID, Host: pendingGame.Host, HostSyncPeriod: 1000, TimestepPeriod: 100, Timestep: 0}}
}

func (tg *TronGame) Init() {
	width := 80
	height := 24
	collisions := make([][]bool, width)
	for i := range collisions {
		collisions[i] = make([]bool, height)
	}
	tg.GameState = TronGameState{width, height, false, collisions}

	clientState := make(map[string]TronClientState)
	for i, player := range tg.PlayerList {
		x := 40 + rand.Intn(10) - 5
		y := 12 + rand.Intn(10) - 5
		clientState[player.Client.ID] = TronClientState{tg.Timestep, true, TRON_COLORS[i], []int{x}, []int{y}, x, y, TronDown}
	}

	tg.ClientStates = clientState
	tg.state = clientState[tg.Me]
	tg.renderCh = make(chan int)

	go func() {
		for {
			tg.updateSelf()
			time.Sleep(time.Duration(tg.TimestepPeriod * int(time.Millisecond)))
			tg.updateOthers()
			mgr.view.Render(mgr.screen)
			mgr.screen.Show()
		}
	}()
}

func (tg *TronGame) ProcessEvent(ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		tg.ProcessEventKey(ev)

	}
}

func (tg *TronGame) ProcessEventKey(ev *tcell.EventKey) {
	key := ev.Key()
	switch key {
	case tcell.KeyUp:
		tg.state.direction = TronUp
	case tcell.KeyRight:
		tg.state.direction = TronRight
	case tcell.KeyDown:
		tg.state.direction = TronDown
	case tcell.KeyLeft:
		tg.state.direction = TronLeft
	}
}

func ProcessMessage(from *Client, p interface{}) {
	return
}

func (tg *TronGame) Render(s *Screen) {
	style := tcell.StyleDefault.Background(tcell.ColorDarkGreen).Foreground(tcell.ColorWhite)

	s.DrawLine(0, 0, tg.GameState.width, 0, style, true)
	s.DrawLine(tg.GameState.width, 0, tg.GameState.width, tg.GameState.height, style, true)
	s.DrawLine(0, tg.GameState.height, tg.GameState.width, tg.GameState.height, style, true)
	s.DrawLine(0, 0, 0, tg.GameState.height, style, true)

	for _, client := range tg.ClientStates {
		for i := 0; i < len(client.pathX); i++ {
			s.SetContent(client.pathX[i], client.pathY[i], '*', nil, style)
		}
		if tg.state.alive {
			s.SetContent(client.x, client.y, '😎', nil, style)
		} else {
			s.SetContent(client.x, client.y, '😵', nil, style)
		}

	}
}

func (tg *TronGame) updateSelf() {
	// tg.state.direction = (tg.ClientStates[tg.Me].direction + 1 % 4)
	if !tg.state.alive {
		return
	}

	x := tg.state.x
	y := tg.state.y
	switch tg.state.direction {
	case TronUp:
		y -= 1
	case TronRight:
		x += 1
	case TronDown:
		y += 1
	case TronLeft:
		x -= 1
	}
	if tg.isOutOfBounds(x, y) || tg.GameState.collisions[x][y] {
		tg.state.alive = false
	}

	tg.setCollision(x, y)
	tg.state.x = x
	tg.state.y = y
	// fmt.Print(tg.state)
	tg.state.pathX = append(tg.state.pathX, tg.state.x)
	tg.state.pathY = append(tg.state.pathY, tg.state.y)
	tg.Timestep += 1
	tg.sendClientUpdate(tg.state)
}

func (tg *TronGame) isOutOfBounds(x int, y int) bool {
	return x < 1 || x >= tg.GameState.width || y < 1 || y >= tg.GameState.height
}

func (tg *TronGame) setCollision(x int, y int) {
	if !tg.isOutOfBounds(x, y) {
		tg.GameState.collisions[x][y] = true
	}
}

func (tg *TronGame) updateOthers() {
	for ip, state := range tg.ClientStates {
		if ip != tg.Me && state.timestep < tg.Timestep {
			tg.ClientStates[ip] = tg.clientPredict(state, tg.Timestep)
		}
	}
}

func (tg *TronGame) clientPredict(state TronClientState, targetTimestep int) TronClientState {
	if targetTimestep <= state.timestep {
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
	// fmt.Print(newPathX, newPathY)
	state.pathX = append(state.pathX, newPathX...)
	state.pathY = append(state.pathY, newPathY...)
	state.x = newPathX[len(newPathX)-1]
	state.y = newPathY[len(newPathY)-1]
	return state
}
