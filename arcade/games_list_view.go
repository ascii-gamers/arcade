package arcade

import (
	"encoding"
	"fmt"
	"sort"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type GamesListView struct {
	View

	mu sync.RWMutex

	lobbies      map[string]*Lobby
	selectedRow  int
	stopTickerCh chan bool

	helloMessageTimes map[string]time.Time
	lastTimeRefreshed int
}

var header = []string{
	"▄▀█ █▀ █▀▀ █ █   ▄▀█ █▀█ █▀▀ ▄▀█ █▀▄ █▀▀",
	"█▀█ ▄█ █▄▄ █ █   █▀█ █▀▄ █▄▄ █▀█ █▄▀ ██▄",
}

var footer = []string{
	"[C]reate new lobby      [J]oin selected lobby by IP address",
}

var glv_join_box = ""
var selectedLobbyKey = ""
var err_msg = ""

const (
	nameColX    = 4
	gameColX    = 30
	playersColX = 40
	pingColX    = 70

	tableX1 = 3
	tableY1 = 7
	tableX2 = 76
	tableY2 = 18

	joinbox_X1 = 7
	joinbox_Y1 = 9
	joinbox_X2 = 72
	joinbox_Y2 = 15
)

var glv_code_input_string = ""
var glv_code = ""

func NewGamesListView() *GamesListView {
	view := &GamesListView{
		stopTickerCh:      make(chan bool),
		lobbies:           make(map[string]*Lobby),
		helloMessageTimes: make(map[string]time.Time),
	}

	ticker := time.NewTicker(time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				// send out lobbyinfo
				view.mu.Lock()
				view.lastTimeRefreshed = (view.lastTimeRefreshed + 1) % 4
				lastTimeRefreshed := view.lastTimeRefreshed
				view.mu.Unlock()
				if lastTimeRefreshed == 0 {
					view.SendHelloMessages()
				}
				arcade.ViewManager.RequestRender()

			case <-view.stopTickerCh:
				ticker.Stop()
				return
			}
		}
	}()

	return view
}

func (v *GamesListView) Init() {
	v.SendHelloMessages()
}

func (v *GamesListView) SendHelloMessages() {
	// Scan LAN for lobbies
	if arcade.LAN {
		go arcade.Server.ScanLAN()
	}

	// Send hello messages to everyone we find
	arcade.Server.Network.ClientsRange(func(client *Client) bool {
		if client.Distributor {
			return true
		}

		arcade.Server.Network.Send(client, NewHelloMessage())

		v.mu.Lock()
		v.helloMessageTimes[client.ID] = time.Now()
		v.mu.Unlock()

		return true
	})
}

func (v *GamesListView) ProcessEvent(evt interface{}) {
	switch evt := evt.(type) {
	case *ClientConnectEvent:
		if client, ok := arcade.Server.Network.GetClient(evt.ClientID); ok {
			arcade.Server.Network.Send(client, NewHelloMessage())

			v.mu.Lock()
			v.helloMessageTimes[client.ID] = time.Now()
			v.mu.Unlock()
		}
	case *tcell.EventKey:
		if len(err_msg) > 0 {
			err_msg = ""
			glv_join_box = ""
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
			if glv_join_box != "" {
				if len(glv_code_input_string) > 0 {
					glv_code_input_string = glv_code_input_string[:len(glv_code_input_string)-1]
				}
			}
		case tcell.KeyEnter:
			if glv_join_box == "join_code" {
				if len(glv_code_input_string) == 4 {
					glv_code = glv_code_input_string
					selectedLobby := v.lobbies[selectedLobbyKey]
					host, _ := arcade.Server.Network.GetClient(selectedLobby.HostID)

					go arcade.Server.Network.Send(host, NewJoinMessage(glv_code, arcade.Server.ID, selectedLobby.ID))
				} else {
					glv_join_box = "join_code"
					err_msg = "Code must be four characters long."
				}

			}
		case tcell.KeyRune:
			if glv_join_box == "" {
				switch evt.Rune() {
				case 'c':
					glv_join_box = ""
					arcade.ViewManager.SetView(NewLobbyCreateView())
				case 't':
					// tg := CreateGame("bruh", false, "Tron", 8, "1", "1")
					// tg.AddPlayer(&Player{Client: *NewClient("addr1"), Username: "bob", Status:  "chillin",Host: true})
					// mgr.SetView(tg)
				case 'j':
					if len(v.lobbies) != 0 {
						v.mu.RLock()

						keys := make([]string, 0, len(v.lobbies))

						for k := range v.lobbies {
							keys = append(keys, k)
						}
						sort.Strings(keys)

						selectedLobbyKey = keys[v.selectedRow]
						selectedLobby := v.lobbies[keys[v.selectedRow]]
						if selectedLobby.Private {
							glv_join_box = "join_code"
						} else {
							host, _ := arcade.Server.Network.GetClient(selectedLobby.HostID)

							go arcade.Server.Network.Send(host, NewJoinMessage("", arcade.Server.ID, selectedLobby.ID))
						}
						v.mu.RUnlock()

					}

				}
			} else {
				if len(glv_code_input_string) < 4 {
					glv_code_input_string += string(evt.Rune())
				}
			}
		}
	}
}

func (v *GamesListView) ProcessMessage(from *Client, p interface{}) interface{} {
	switch p := p.(type) {
	case LobbyInfoMessage:
		v.mu.Lock()
		p.Lobby.Ping = int(time.Since(v.helloMessageTimes[from.ID]).Milliseconds())
		v.lobbies[p.Lobby.ID] = p.Lobby
		v.mu.Unlock()

		arcade.ViewManager.RequestRender()
	case JoinReplyMessage:
		if p.Error == OK {
			arcade.lobbyMux.Lock()
			arcade.Lobby = p.Lobby
			arcade.lobbyMux.Unlock()

			err_msg = ""
			glv_join_box = ""
			glv_code_input_string = ""
			arcade.ViewManager.SetView(NewLobbyView())

			arcade.Server.BeginHeartbeats(p.Lobby.HostID)
		} else if p.Error == ErrWrongCode {
			err_msg = "Wrong join code."
		} else if p.Error == ErrCapacity {
			err_msg = "Game is now full."
		}
	case LobbyEndMessage:
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
	// fmt.Println("HELP")
	if glv_join_box == "" && len(glv_code_input_string) > 0 {
		s.Clear()
		glv_code_input_string = ""
	}
	// s.Clear()

	width, height := s.displaySize()

	// if glv_join_box == "" {
	// 	sty_black := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorBlack)
	// 	s.DrawBox(joinbox_X1, joinbox_Y1, joinbox_X2, joinbox_Y2, sty_black, true)

	// }

	// Green text on default background
	sty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen)

	// Draw ASCII ARCADE header
	headerX := (width - utf8.RuneCountInString(header[0])) / 2
	s.DrawText(headerX, 1, sty, header[0])
	s.DrawText(headerX, 2, sty, header[1])

	// Draw box surrounding games list
	s.DrawBox(tableX1-1, 4, tableX2+1, tableY2+1, sty, true)

	// Draw footer with navigation keystrokes
	s.DrawText((width-len(footer[0]))/2, height-2, sty, footer[0])

	v.mu.Lock()
	countdownMsg := fmt.Sprintf("Refreshing in %d", 4-v.lastTimeRefreshed)
	// Draw countdown

	if v.lastTimeRefreshed == 0 {
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
	s.DrawLine(3, 6, tableX2, 6, sty, true)
	s.DrawText(2, 6, sty, "╠")
	s.DrawText(width-3, 6, sty, "╣")

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

	if glv_join_box != "" {

		selectedLobby := v.lobbies[selectedLobbyKey]
		// Draw box surrounding games list
		s.DrawBox(joinbox_X1, joinbox_Y1, joinbox_X2, joinbox_Y2, sty, true)

		joinheader := "Joining private game " + selectedLobby.Name
		s.DrawText((width-len(joinheader))/2, joinbox_Y1+1, sty, "Joining private game ")
		s.DrawText((width-len(joinheader))/2+len(joinheader)-len(selectedLobby.Name), joinbox_Y1+1, sty_bold, selectedLobby.Name)
		codeHeader := "Enter code: "
		s.DrawText((width-len(codeHeader)-5)/2, joinbox_Y1+2, sty, codeHeader)
		s.DrawText((width-len(codeHeader)-5)/2+len(codeHeader), joinbox_Y1+2, sty_bold, glv_code_input_string)
		s.DrawText((width-len(codeHeader)-5)/2+len(codeHeader)+len(glv_code_input_string), joinbox_Y1+2, sty_bold, "    ")
		if len(err_msg) > 0 {
			shortString := err_msg + " Press any key to continue."
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
