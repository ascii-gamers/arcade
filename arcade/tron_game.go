package arcade

import (
	"log"
	"time"

	"github.com/gdamore/tcell/v2"
)

type TronDirection int64 
const (
	TronUp TronDirection = iota
	TronRight
	TronDown
	TronLeft
)


type TronGameState struct {
	width int
	height int
	ended bool
}

type TronClientState struct {
	timestep int
	alive bool 
	color string 
	pathX []int
	pathY []int
	x int 
	y int
	direction TronDirection
}

type TronGame struct {
	Game[TronGameState, TronClientState]
	state TronClientState
	s tcell.Screen
	style tcell.Style
}

func (tg *TronGame) start() {
	tg.initScreen()
	tg.initState()

	go func () {
		for {
			tg.updateSelf()
			time.Sleep(time.Duration(tg.timestepPeriod))
			tg.updateOthers()
			tg.render()
		}
	}()
}
func (tg *TronGame) initState() {
	tg.clientIps = []string{"1","2"}
	tg.clientStates = map[string]TronClientState{"1": {tg.timestep, true, "blue", []int{},[]int{},50, 50, TronDown}, "2": {tg.timestep, true, "blue", []int{},[]int{},40, 40, TronLeft}}
}


func (tg *TronGame) initScreen() {
	tg.Game.start()
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}

	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	s.SetStyle(defStyle)
	s.Clear()
	tg.s = s
}

func (tg *TronGame) updateSelf() {
	tg.state.direction = (tg.clientStates[tg.me].direction + 1 % 4)
	switch tg.state.direction {
	case TronUp:
		tg.state.y -= 1
	case TronRight:
		tg.state.x += 1
	case TronDown:
		tg.state.y += 1
	case TronLeft:
		tg.state.x -= 1
	}
	tg.state.pathX = append(tg.state.pathX, tg.state.x)
	tg.state.pathY = append(tg.state.pathY, tg.state.y)
	tg.timestep += 1
	tg.sendClientUpdate(tg.state)
}

func (tg *TronGame) updateOthers() {
	for ip, state := range tg.clientStates {
		if state.timestep < tg.timestep {
			tg.clientStates[ip] = tg.clientPredict(state, tg.timestep)
		}
	}
}

func (tg *TronGame) clientPredict(state TronClientState, targetTimestep int) TronClientState {
	if targetTimestep < state.timestep {
		return state 
	}
	delta := targetTimestep - state.timestep
	newPathX := []int{}
	newPathY := []int{}
	lastX := state.pathX[len(state.pathX) - 1]
	lastY := state.pathX[len(state.pathX) - 1]
	for i := 1; i <= delta; i++ {
		switch state.direction {
		case TronUp:
			newPathX = append(newPathX, lastX)
			newPathY = append(newPathY, lastY - i)
		case TronRight:
			newPathX = append(newPathX, lastX + i)
			newPathY = append(newPathY, lastY)
		case TronDown:
			newPathX = append(newPathX, lastX)
			newPathY = append(newPathY, lastY + i)
		case TronLeft:
			newPathX = append(newPathX, lastX - i)
			newPathY = append(newPathY, lastY)
		}
	}
	state.pathX = append(state.pathX, newPathX...)
	state.pathY = append(state.pathY, newPathY...)
	state.x = newPathX[len(newPathX) - 1]
	state.y = newPathY[len(newPathY) - 1]
	return state
}

func (tg TronGame) render() {
	for _, client := range tg.clientStates {

		for i := 0; i < len(client.pathX); i++ {
			tg.s.SetContent(client.pathX[i], client.pathY[i], '*', nil, tg.style)
		}
		tg.s.SetContent(client.x, client.y, 'F', nil, tg.style)
	}
}

