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
	Width      int
	Height     int
	Ended      bool
	Collisions []byte
}

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
var processedInp = false

func (tg *TronGameView) Init() {
	width, height := arcade.ViewManager.screen.displaySize()
	// collisions := make([][]bool, width)
	// for i := range collisions {
	// 	collisions[i] = make([]bool, height)
	// }
	collisions := make([]byte, int(math.Ceil(float64(width*height)/2)))
	// tg.GameState.Collisions = collisions
	tg.GameState = TronGameState{width, height, false, collisions}

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
		for !tg.GameState.Ended {
			tg.Timestep += 1
			processedInp = false
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
	}
	return nil
}

func (tg *TronGameView) Render(s *Screen) {

	// defaultStyle := tcell.StyleDefault.Background(tcell.ColorTeal).Foreground(tcell.ColorTeal)

	s.ClearContent()

	// displayWidth, displayHeight := s.displaySize()
	// s.DrawBox(0, 0, displayWidth-1, displayHeight-1, defaultStyle, true)

	for row := 0; row < tg.GameState.Width; row++ {
		for col := 0; col < tg.GameState.Height; col++ {
			if ok, playerNum := tg.getCollision(row, col); ok && playerNum >= 0 {
				style := tcell.StyleDefault.Background(tcell.ColorNames[TRON_COLORS[playerNum]])
				s.DrawText(row, col, style, " ")
			}

		}
	}

	for _, client := range tg.ClientStates {
		// style := tcell.StyleDefault.Background(tcell.ColorNames[client.Color])
		hStyle := tcell.StyleDefault.Foreground(tcell.ColorNames[client.Color]) //.Foreground(tcell.ColorWhite)
		// vStyle := tcell.StyleDefault.Background(tcell.ColorNames[client.Color])
		for i := 0; i < len(client.PathX)-1; i++ {
			// s.DrawText(client.PathX[i], client.PathY[i], style, " ")
			// if math.Abs(float64(client.PathX[i+1]-client.PathX[i])) > 0 {
			// 	s.DrawText(client.PathX[i], client.PathY[i], hStyle, "■")
			// } else if math.Abs(float64(client.PathY[i+1]-client.PathY[i])) > 0 {
			// 	s.DrawText(client.PathX[i], client.PathY[i], vStyle, " ")
			// }

			// if math.Abs(float64(client.PathX[i+1]-client.PathX[i])) > 0 && i < len(client.PathX)-2 {
			// 	s.DrawText(client.PathX[i], client.PathY[i], style, " ")
			// } else if math.Abs(float64(client.PathY[i+1]-client.PathY[i])) > 0 {
			// 	s.DrawText(client.PathX[i], client.PathY[i], style, " ")
			// }

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

	tg.setCollision(state.X, state.Y, state.PlayerNum)
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

	// for i := 0; i < delta-1; i++ {
	// 	newPathX = append(newPathX, lastX)
	// 	newPathY = append(newPathY, lastY)
	// }

	// switch state.Direction {
	// case TronUp:
	// 	newPathX = append(newPathX, lastX)
	// 	newPathY = append(newPathY, lastY-1)
	// case TronRight:
	// 	newPathX = append(newPathX, lastX+1)
	// 	newPathY = append(newPathY, lastY)
	// case TronDown:
	// 	newPathX = append(newPathX, lastX)
	// 	newPathY = append(newPathY, lastY+1)
	// case TronLeft:
	// 	newPathX = append(newPathX, lastX-1)
	// 	newPathY = append(newPathY, lastY)
	// }

	state.Timestep = targetTimestep
	state.PathX = append(state.PathX, newPathX...)
	state.PathY = append(state.PathY, newPathY...)
	state.X = newPathX[len(newPathX)-1]
	state.Y = newPathY[len(newPathY)-1]

	if tg.shouldDie(state) {
		state = tg.die(state)
	}
	tg.setCollision(state.X, state.Y, state.PlayerNum)

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

func (g *Game[GS, CS]) sendClientUpdate(update CS) {
	// g.ClientStates[g.Me] = update
	clientUpdate := &ClientUpdateMessage[CS]{Message: Message{Type: "client_update"}, Id: g.Me, Update: update}

	for clientId := range g.ClientStates {
		if client, ok := arcade.Server.Network.GetClient(clientId); ok && clientId != g.Me {
			arcade.Server.Network.Send(client, clientUpdate)
		}
	}
}

func (g *Game[GS, CS]) sendGameUpdate() {
	if g.Me != g.HostID {
		return
	}

	arcade.Server.RLock()
	defer arcade.Server.RUnlock()

	for clientId := range g.ClientStates {
		if client, ok := arcade.Server.Network.GetClient(clientId); ok && clientId != g.Me {
			data := &GameUpdateMessage[GS, CS]{Message{Type: "game_update"}, g.GameState, g.ClientStates, lastReceivedInp}
			arcade.Server.Network.Send(client, data)
		}
	}
}

func (tg *TronGameView) recalculateCollisions() {
	width, height := arcade.ViewManager.screen.displaySize()
	collisions := make([]byte, int(math.Ceil(float64(width*height)/2)))
	tg.GameState.Collisions = collisions
	for _, player := range tg.ClientStates {
		for i := 0; i < len(player.PathX); i++ {
			tg.setCollision(player.PathX[i], player.PathY[i], player.PlayerNum)
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
	collides, _ := tg.getCollision(player.X, player.Y)
	return tg.isOutOfBounds(player.X, player.Y) || collides //tg.GameState.Collisions[player.X][player.Y]
}

func (tg *TronGameView) die(player TronClientState) TronClientState {
	player.Alive = false
	return player
}

func (tg *TronGameView) isOutOfBounds(x int, y int) bool {
	return x <= 0 || x >= tg.GameState.Width-1 || y <= 0 || y >= tg.GameState.Height-1
}

// func (tg *TronGameView) setCollision(x int, y int) {
// 	width, _ := arcade.ViewManager.screen.displaySize()
// 	if !tg.isOutOfBounds(x, y) {
// 		ind := y*width + x
// 		tg.GameState.Collisions[ind/8] |= 1 << (ind % 8)
// 	}
// }

// func (tg *TronGameView) getCollision(x int, y int) bool {
// 	width, _ := arcade.ViewManager.screen.displaySize()
// 	if !tg.isOutOfBounds(x, y) {
// 		ind := y*width + x
// 		if tg.GameState.Collisions[ind/8]>>(ind%8)&1 == 1 {
// 			return true
// 		} else {
// 			return false
// 		}
// 	}
// 	return true
// }

func (tg *TronGameView) setCollision(x int, y int, playerNum int) {

	width, _ := arcade.ViewManager.screen.displaySize()
	if !tg.isOutOfBounds(x, y) && playerNum < 8 {
		ind := y*width + x
		tg.GameState.Collisions[ind/2] |= byte(playerNum<<1+1) << ((ind % 2) * 4)
	}
}

func (tg *TronGameView) getCollision(x int, y int) (bool, int) {
	width, _ := arcade.ViewManager.screen.displaySize()
	if !tg.isOutOfBounds(x, y) {
		ind := y*width + x
		offset := ((ind % 2) * 4)
		coll := tg.GameState.Collisions[ind/2] >> offset

		if coll&1 == 1 {
			// fmt.Println("B: ", coll, coll>>1, int((coll>>1)&7))
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

func (v *TronGameView) GetHeartbeatMetadata() encoding.BinaryMarshaler {
	return nil
}
