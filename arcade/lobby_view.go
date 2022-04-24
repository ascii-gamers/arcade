package arcade

import (
	"sync"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type LobbyView struct {
	View

	mu sync.RWMutex
	servers     map[string]LobbyInfoMessage
	selectedRow int
	name string
	game int
	players int 
	
}

var create_game_header = []string{
	"| █▀▀ █▀█ █▀▀ ▄▀█ ▀█▀ █▀▀   █▀▀ ▄▀█ █▄█ █▀▀ |",
	"| █▄▄ █▀▄ ██▄ █▀█  █  ██▄   █▄█ █▀█ █ █ ██▄ |",										  
}

var game_footer = []string{
	"[P]ublish game       [C]ancel",
}

func NewLobbyView() *LobbyView {
	return &LobbyView{}
}

func (v *LobbyView) Init() {
}

func (v *LobbyView) ProcessEvent(evt tcell.Event) {
	switch evt := evt.(type) {
	case *tcell.EventKey:
		switch evt.Key() {
		case tcell.KeyDown:
			v.selectedRow++

			v.mu.RLock()
			if v.selectedRow > len(v.servers)-1 {
				v.selectedRow = len(v.servers) - 1
			}
			v.mu.RUnlock()
		case tcell.KeyUp:
			v.selectedRow--

			if v.selectedRow < 0 {
				v.selectedRow = 0
			}
		case tcell.KeyRune:
			switch evt.Rune() {
			case 'c':
				mgr.SetView(NewGamesListView())
			case 'p':
				// save things
				// mgr.SetView(NewLobbyView())
			}
		}
	}
}

func (v *LobbyView) ProcessPacket(p interface{}) interface{} {
	switch p.(type) {
	case HelloMessage:
		return NewLobbyInfoMessage(server.Addr)
	}

	return nil
}

func (v *LobbyView) Render(s *Screen) {
	width, height := s.Size()

	// Green text on default background
	sty := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorLightSlateGray)
	// Dark blue text on light gray background
	sty_game := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorMidnightBlue)

	// Draw ASCII ARCADE header
	headerX := (width - utf8.RuneCountInString(header[0])) / 2
	s.DrawText(headerX, 1, sty, header[0])
	s.DrawText(headerX, 2, sty, header[1])

	// draw create game header
	header2X := (width - utf8.RuneCountInString(create_game_header[0])) / 2
	s.DrawText(header2X, 4, sty_game, create_game_header[0])
	s.DrawText(header2X, 5, sty_game, create_game_header[1])

	// Draw box surrounding games list
	// s.DrawBox(tableX1-1, 5, tableX2+1, tableY2+1, sty, true)

	// Draw footer with navigation keystrokes
	s.DrawText((width-len(game_footer[0]))/2, height-2, sty_game, game_footer[0])

	// Draw column headers
	
	// s.DrawText(gameColX, 5, sty, "GAME")
	// s.DrawText(playersColX, 5, sty, "PLAYERS")
	// s.DrawText(pingColX, 5, sty, "PING")

	// // Draw border below column headers
	// s.DrawLine(3, 6, tableX2, 6, sty, true)
	// s.DrawText(2, 6, sty, "╠")
	// s.DrawText(width-3, 6, sty, "╣")

	// // Draw selected row
	// selectedSty := tcell.StyleDefault.Background(tcell.ColorDarkGreen).Foreground(tcell.ColorWhite)

	// i := 0
	// v.mu.RLock()

	// for _, lobby := range v.servers {
	// 	if lobby.IP == server.Addr {
	// 		continue
	// 	}

	// 	y := tableY1 + i
	// 	rowSty := sty

	// 	if i == v.selectedRow {
	// 		rowSty = selectedSty
	// 	}

	// 	name := lobby.IP
	// 	game := "Pong"
	// 	players := "1/2"
	// 	ping := "25ms"

	// 	s.DrawEmpty(tableX1, y, nameColX-1, y, rowSty)
	// 	s.DrawText(nameColX, y, rowSty, name)
	// 	s.DrawEmpty(nameColX+len(name), y, gameColX-1, y, rowSty)
	// 	s.DrawText(gameColX, y, rowSty, game)
	// 	s.DrawEmpty(gameColX+len(game), y, playersColX-1, y, rowSty)
	// 	s.DrawText(playersColX, y, rowSty, players)
	// 	s.DrawEmpty(playersColX+len(players), y, pingColX-1, y, rowSty)
	// 	s.DrawText(pingColX, y, rowSty, ping)
	// 	s.DrawEmpty(pingColX+len(ping), y, tableX2, y, rowSty)

	// 	i++
	// }

	// v.mu.RUnlock()
}
