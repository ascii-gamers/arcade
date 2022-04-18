package arcade

import (
	"sync"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type GamesListView struct {
	View

	servers     sync.Map
	selectedRow int
}

var header = []string{
	"▄▀█ █▀ █▀▀ █ █   ▄▀█ █▀█ █▀▀ ▄▀█ █▀▄ █▀▀",
	"█▀█ ▄█ █▄▄ █ █   █▀█ █▀▄ █▄▄ █▀█ █▄▀ ██▄",
}

var footer = []string{
	"[C]reate new lobby      [J]oin lobby by IP address",
}

const (
	nameColX    = 4
	gameColX    = 30
	playersColX = 40
	pingColX    = 70

	tableX1 = 3
	tableY1 = 7
	tableX2 = 76
	tableY2 = 18
)

func NewGamesListView() *GamesListView {
	return &GamesListView{
		selectedRow: tableY1,
	}
}

func (v *GamesListView) Init() {
	go server.connectToNextOpenPort()
}

func (v *GamesListView) ProcessEvent(evt tcell.Event) {
	switch evt := evt.(type) {
	case *tcell.EventKey:
		switch evt.Key() {
		case tcell.KeyDown:
			v.selectedRow++

			if v.selectedRow > tableY2 {
				v.selectedRow = tableY2
			}
		case tcell.KeyUp:
			v.selectedRow--

			if v.selectedRow <= tableY1 {
				v.selectedRow = tableY1
			}
		case tcell.KeyRune:
			switch evt.Rune() {
			case 'c':
				mgr.SetView(NewLobbyView())
			}
		}
	}
}

func (v *GamesListView) ProcessPacket(p interface{}) interface{} {
	switch p := p.(type) {
	case LobbyInfoMessage:
		v.servers.Store(p.IP, p)
	}

	return nil
}

func (v *GamesListView) Render(s *Screen) {
	width, height := s.Size()

	// Green text on default background
	sty := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorGreen)

	// Draw ASCII ARCADE header
	headerX := (width - utf8.RuneCountInString(header[0])) / 2
	s.DrawText(headerX, 1, sty, header[0])
	s.DrawText(headerX, 2, sty, header[1])

	// Draw box surrounding games list
	s.DrawBox(tableX1-1, 4, tableX2+1, tableY2+1, sty, true)

	// Draw footer with navigation keystrokes
	s.DrawText((width-len(footer[0]))/2, height-2, sty, footer[0])

	// Draw column headers
	s.DrawText(nameColX, 5, sty, "NAME")
	s.DrawText(gameColX, 5, sty, "GAME")
	s.DrawText(playersColX, 5, sty, "PLAYERS")
	s.DrawText(pingColX, 5, sty, "PING")

	// Draw border below column headers
	s.DrawLine(3, 6, tableX2, 6, sty, true)
	s.DrawText(2, 6, sty, "╠")
	s.DrawText(width-3, 6, sty, "╣")

	// Draw selected row
	selectedSty := tcell.StyleDefault.Background(tcell.ColorGray).Foreground(tcell.ColorGreen)

	i := 0
	v.servers.Range(func(ip, value any) bool {
		lobby := value.(LobbyInfoMessage)

		if lobby.IP == server.Addr {
			return true
		}

		row := tableY1 + i
		rowSty := sty

		if row == v.selectedRow {
			rowSty = selectedSty
			// s.DrawEmpty(tableX1, row, tableX2, row, rowSty)
		}

		s.DrawText(nameColX, row, rowSty, lobby.IP)
		s.DrawText(gameColX, row, rowSty, "Pong")
		s.DrawText(playersColX, row, rowSty, "1/2")
		s.DrawText(pingColX, row, rowSty, "25ms")

		i++
		return true
	})

	// for row := tableY1; row <= tableY2; row++ {
	// 	if row == v.selectedRow {
	// 		s.DrawEmpty(tableX1, row, tableX2, row, selectedSty)
	// 	} else {
	// 		s.DrawEmpty(tableX1, row, tableX2, row, sty)
	// 	}
	// }
}
