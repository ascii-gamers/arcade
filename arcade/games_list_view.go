package arcade

import (
	"arcade/arcade/multicast"
	"arcade/arcade/net"
	"encoding"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
)

type GamesListView struct {
	View
	mgr *ViewManager

	mu sync.RWMutex

	lobbies      map[string]*Lobby
	selectedRow  int
	stopTickerCh chan bool

	lastTimeRefreshed int

	glv_join_box          string
	selectedLobbyKey      string
	err_msg               string
	glv_code_input_string string
	glv_code              string
}

var footer = []string{
	"[C]reate new lobby      [J]oin selected lobby",
}

// const (
// 	nameColX    = 4
// 	gameColX    = 30
// 	playersColX = 40
// 	pingColX    = 70

// 	tableX1 = 3
// 	tableY1 = 7
// 	tableX2 = 76
// 	tableY2 = 18

// 	joinbox_X1 = 7
// 	joinbox_Y1 = 9
// 	joinbox_X2 = 72
// 	joinbox_Y2 = 15
// )

func NewGamesListView(mgr *ViewManager) *GamesListView {
	return &GamesListView{
		mgr:               mgr,
		stopTickerCh:      make(chan bool),
		lobbies:           make(map[string]*Lobby),
		lastTimeRefreshed: 3,
	}
}

func (v *GamesListView) Init() {
	ticker := time.NewTicker(time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				v.mu.Lock()
				v.lastTimeRefreshed = (v.lastTimeRefreshed + 1) % 6

				if v.lastTimeRefreshed == 0 {
					// Send out hello messages when the timer hits zero
					go v.SendHelloMessages()
				}

				v.mu.Unlock()
				v.mgr.RequestRender()
			case <-v.stopTickerCh:
				ticker.Stop()
				return
			}
		}
	}()

	go v.SendHelloMessages()
}

func (v *GamesListView) SendHelloMessages() {
	// Scan LAN for lobbies
	go multicast.Discover(arcade.Server.Addr, arcade.Server.ID, arcade.Port)

	// Send hello messages to everyone we find
	arcade.Server.Network.ClientsRange(func(client *net.Client) bool {
		client.RLock()
		if (client.State != net.Connected && client.State != net.Connecting) || client.Distributor {
			client.RUnlock()
			return true
		}
		client.RUnlock()

		go v.QueryClient(client)
		return true
	})
}

// QueryClient sends a HelloMessage to the client and waits for a reply. If a
// LobbyInfoMessage is received, the client immediately re-renders the view
// with the new lobby included.
func (v *GamesListView) QueryClient(client *net.Client) {
	start := time.Now()
	res, err := arcade.Server.Network.SendAndReceive(client, NewHelloMessage())
	end := time.Now()

	p, ok := res.(*LobbyInfoMessage)

	if !ok || err != nil {
		return
	}

	v.mu.Lock()
	p.Lobby.Ping = int(end.Sub(start).Milliseconds())
	v.lobbies[p.Lobby.ID] = p.Lobby
	v.mu.Unlock()

	v.mgr.RequestRender()
}

func (v *GamesListView) ProcessEvent(evt interface{}) {
	switch evt := evt.(type) {
	case *ClientConnectedEvent:
		if client, ok := arcade.Server.Network.GetClient(evt.ClientID); ok {
			go v.QueryClient(client)
		}
	case *ClientDisconnectedEvent:
		v.mu.Lock()
		delete(v.lobbies, evt.ClientID)
		v.mu.Unlock()

		v.mgr.RequestRender()
	case *tcell.EventKey:
		if len(v.err_msg) > 0 {
			v.err_msg = ""
			v.glv_join_box = ""
			return
		}
		switch evt.Key() {
		case tcell.KeyDown:
			v.selectedRow++

			v.mu.RLock()
			if v.selectedRow > len(v.lobbies)-1 {
				v.selectedRow = len(v.lobbies) - 1
			}
			v.mu.RUnlock()
		case tcell.KeyUp:
			v.selectedRow--

			if v.selectedRow < 0 {
				v.selectedRow = 0
			}
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if v.glv_join_box != "" {
				if len(v.glv_code_input_string) > 0 {
					v.glv_code_input_string = v.glv_code_input_string[:len(v.glv_code_input_string)-1]
				}
			}
		case tcell.KeyEnter:
			if v.glv_join_box == "join_code" {
				if len(v.glv_code_input_string) == 4 {
					v.glv_code = v.glv_code_input_string
					selectedLobby := v.lobbies[v.selectedLobbyKey]
					host, _ := arcade.Server.Network.GetClient(selectedLobby.HostID)

					go arcade.Server.Network.Send(host, NewJoinMessage(v.glv_code, arcade.Server.ID, selectedLobby.ID))
				} else {
					v.glv_join_box = "join_code"
					v.err_msg = "Code must be four characters long."
				}

			}
		case tcell.KeyRune:
			if v.glv_join_box == "" {
				switch evt.Rune() {
				case 'c':
					v.glv_join_box = ""
					v.mgr.SetView(NewCreateLobbyView(v.mgr))
				case 'j':
					if len(v.lobbies) != 0 {
						v.mu.RLock()

						keys := make([]string, 0, len(v.lobbies))

						for k := range v.lobbies {
							keys = append(keys, k)
						}
						sort.Strings(keys)

						v.selectedLobbyKey = keys[v.selectedRow]
						selectedLobby := v.lobbies[keys[v.selectedRow]]
						if selectedLobby.Private {
							v.glv_join_box = "join_code"
						} else {
							host, _ := arcade.Server.Network.GetClient(selectedLobby.HostID)

							go arcade.Server.Network.Send(host, NewJoinMessage("", arcade.Server.ID, selectedLobby.ID))
						}
						v.mu.RUnlock()

					}

				}
			} else {
				if len(v.glv_code_input_string) < 4 {
					v.glv_code_input_string += string(evt.Rune())
				}
			}
		}
	}
}

func (v *GamesListView) ProcessMessage(from *net.Client, p interface{}) interface{} {
	switch p := p.(type) {
	case *JoinReplyMessage:
		if p.Error == OK {
			v.mu.Lock()
			v.err_msg = ""
			v.glv_join_box = ""
			v.glv_code_input_string = ""
			v.mu.Unlock()

			v.mgr.SetView(NewLobbyView(v.mgr, p.Lobby))

			arcade.Server.BeginHeartbeats(p.Lobby.HostID)
		} else if p.Error == ErrWrongCode {
			v.mu.Lock()
			v.err_msg = "Wrong join code."
			v.mu.Unlock()
		} else if p.Error == ErrCapacity {
			v.mu.Lock()
			v.err_msg = "Game is now full."
			v.mu.Unlock()
		}
	case *LobbyEndMessage:
		v.mu.Lock()
		delete(v.lobbies, p.LobbyID)
		v.selectedRow--
		if v.selectedRow < 0 {
			v.selectedRow = 0
		}

		v.mu.Unlock()

	}

	return nil
}

func (v *GamesListView) Render(s *Screen) {
	if v.glv_join_box == "" && len(v.glv_code_input_string) > 0 {
		s.Clear()
		v.glv_code_input_string = ""
	}

	width, height := s.displaySize()

	const (
		tableWidth  = 72
		tableHeight = 14
	)

	var (
		tableX1 = (width-tableWidth)/2 - 1
		tableY1 = 7
		tableX2 = width - (width-tableWidth)/2
		tableY2 = tableY1 + tableHeight

		nameColX    = tableX1 + 1
		gameColX    = tableX1 + 27
		playersColX = tableX1 + 37
		pingColX    = tableX1 + 67

		joinbox_X1 = tableX1 + 4
		joinbox_Y1 = tableY1 + 2
		joinbox_X2 = tableX2 - 4
		joinbox_Y2 = tableY2 - 3
	)

	// const (
	// 	nameColX    = 4
	// 	gameColX    = 30
	// 	playersColX = 40
	// 	pingColX    = 70

	// 	tableX1 = 3
	// 	tableY1 = 7
	// 	tableX2 = 76
	// 	tableY2 = 18

	// 	joinbox_X1 = 7
	// 	joinbox_Y1 = 9
	// 	joinbox_X2 = 72
	// 	joinbox_Y2 = 15
	// )

	// Green text on default background
	sty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen)

	// Draw ASCII ARCADE header
	s.DrawBlockText(CenterX(s.GetWidth()), 1, sty, "ASCII ARCADE", false)

	// Draw box surrounding games list
	s.DrawBox(tableX1-1, 4, tableX2+1, tableY2+1, sty, true)

	// Draw footer with navigation keystrokes
	s.DrawText((width-len(footer[0]))/2, height-2, sty, footer[0])

	v.mu.Lock()
	countdownMsg := fmt.Sprintf("Refreshing in %d", 6-v.lastTimeRefreshed)

	if v.lastTimeRefreshed < 3 {
		countdownMsg = "     Refreshing...     "
	}
	v.mu.Unlock()

	s.DrawText((width-len(countdownMsg))/2, height-3, sty, countdownMsg)

	// Draw column headers
	s.DrawText(nameColX, 5, sty, "NAME")
	s.DrawText(gameColX, 5, sty, "GAME")
	s.DrawText(playersColX, 5, sty, "PLAYERS")
	s.DrawText(pingColX, 5, sty, "PING")

	// Draw border below column headers
	s.DrawLine(tableX1, 6, tableX2, 6, sty, true)
	s.DrawText(tableX1-1, 6, sty, "╠")
	s.DrawText(tableX2+1, 6, sty, "╣")

	// Clear screen of potentially old games
	for m := tableY1; m <= tableY2; m++ {
		s.DrawEmpty(tableX1, m, tableX2, m, sty)
	}

	// Draw selected row
	selectedSty := tcell.StyleDefault.Background(tcell.ColorDarkGreen).Foreground(tcell.ColorWhite)
	sty_bold := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorLightGreen)

	i := 0
	v.mu.RLock()

	keys := make([]string, 0, len(v.lobbies))

	for k := range v.lobbies {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, lobbyID := range keys {
		lobby := v.lobbies[lobbyID]
		lobby.mu.RLock()
		y := tableY1 + i
		if y == tableY2+1 {
			lobby.mu.RUnlock()
			break
		}
		rowSty := sty

		if i == v.selectedRow {
			rowSty = selectedSty
		}

		name := lobby.Name
		game := lobby.GameType
		players := fmt.Sprintf("%d/%d", len(lobby.PlayerIDs), lobby.Capacity)
		ping := fmt.Sprintf("%dms", lobby.Ping)
		lobby.mu.RUnlock()

		s.DrawEmpty(tableX1, y, nameColX-1, y, rowSty)
		s.DrawText(nameColX, y, rowSty, name)
		s.DrawEmpty(nameColX+len(name), y, gameColX-1, y, rowSty)
		s.DrawText(gameColX, y, rowSty, game)
		s.DrawEmpty(gameColX+len(game), y, playersColX-1, y, rowSty)
		s.DrawText(playersColX, y, rowSty, players)
		s.DrawEmpty(playersColX+len(players), y, pingColX-1, y, rowSty)
		s.DrawText(pingColX, y, rowSty, ping)
		s.DrawEmpty(pingColX+len(ping), y, tableX2, y, rowSty)
		i++
	}

	if v.glv_join_box != "" {

		selectedLobby := v.lobbies[v.selectedLobbyKey]
		// Draw box surrounding games list
		s.DrawBox(joinbox_X1, joinbox_Y1, joinbox_X2, joinbox_Y2, sty, true)

		joinheader := "Joining private game " + selectedLobby.Name
		s.DrawText((width-len(joinheader))/2, joinbox_Y1+1, sty, "Joining private game ")
		s.DrawText((width-len(joinheader))/2+len(joinheader)-len(selectedLobby.Name), joinbox_Y1+1, sty_bold, selectedLobby.Name)
		codeHeader := "Enter code: "
		s.DrawText((width-len(codeHeader)-5)/2, joinbox_Y1+2, sty, codeHeader)
		s.DrawText((width-len(codeHeader)-5)/2+len(codeHeader), joinbox_Y1+2, sty_bold, v.glv_code_input_string)
		s.DrawText((width-len(codeHeader)-5)/2+len(codeHeader)+len(v.glv_code_input_string), joinbox_Y1+2, sty_bold, "    ")

		if len(v.err_msg) > 0 {
			shortString := v.err_msg + " Press any key to continue."
			s.DrawText((width-len(shortString))/2, joinbox_Y1+4, sty_bold, shortString)
		}
	}
	v.mu.RUnlock()
}

func (v *GamesListView) Unload() {
	v.stopTickerCh <- true
}

func (v *GamesListView) GetHeartbeatMetadata() encoding.BinaryMarshaler {
	return nil
}
