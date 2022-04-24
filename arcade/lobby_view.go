package arcade

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type LobbyView struct {
	View

	mu sync.RWMutex
	servers     map[string]LobbyInfoMessage
	selectedRow int
	newGame *Game
	
}
const game_input_default = "[i] to edit"
var game_user_input = [4]string {"", "", "", ""}
var game_input_categories = [4]string {"NAME", "PRIVATE?", "GAME TYPE", "CAPACITY"}


const (
	lobbyTableX1 = 20
	lobbyTableY1 = 7
	lobbyTableX2 = 59
	lobbyTableY2 = 18
)

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
			if v.selectedRow > len(game_input_categories)-1 {
				v.selectedRow = len(game_input_categories) - 1
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
				mgr.SetView(NewLobbyView())
			case 'i':
				reader := bufio.NewReader(os.Stdin)
				i, _ := reader.ReadString('\n')
				fmt.Println(i)

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
	s.DrawBox(lobbyTableX1-1, 7, lobbyTableX2+1, lobbyTableY2+1, sty_game, true)

	// Draw footer with navigation keystrokes
	s.DrawText((width-len(game_footer[0]))/2, height-2, sty_game, game_footer[0])

	// Draw column headers
	
	// s.DrawText(gameColX, 5, sty, "GAME")
	// s.DrawText(playersColX, 5, sty, "PLAYERS")
	// s.DrawText(pingColX, 5, sty, "PING")

	// // Draw border below column headers
	s.DrawLine(34, 7, 34, tableY2, sty_game, true)
	s.DrawText(34, 7, sty_game, "╦")
	s.DrawText(34, tableY2+1, sty_game, "╩")

	// Draw selected row
	selectedSty := tcell.StyleDefault.Background(tcell.ColorDarkGreen).Foreground(tcell.ColorWhite)

	i := 0
	for index, inputField := range game_input_categories {

		y := lobbyTableY1 + i + 1
		rowSty := sty_game

		if i == v.selectedRow {
			rowSty = selectedSty
		}

		s.DrawEmpty(lobbyTableX1, y, lobbyTableX1, y, rowSty)
		s.DrawText(lobbyTableX1 + 1, y, rowSty, inputField)
		s.DrawEmpty(lobbyTableX1+len(inputField)+1, y, 33, y, rowSty)

		inputString := game_user_input[index]
		if game_user_input[index] == "" {
			inputString = game_input_default
		}

		s.DrawText(35, y, rowSty, inputString)
		s.DrawEmpty(35+len(inputString), y, lobbyTableX2-1, y, rowSty)

		// name := lobby.IP
		// game := "Pong"
		// players := "1/2"
		// ping := "25ms"

		// s.DrawEmpty(tableX1, y, nameColX-1, y, rowSty)
		// s.DrawText(nameColX, y, rowSty, name)
		// s.DrawEmpty(nameColX+len(name), y, gameColX-1, y, rowSty)
		// s.DrawText(gameColX, y, rowSty, game)
		// s.DrawEmpty(gameColX+len(game), y, playersColX-1, y, rowSty)
		// s.DrawText(playersColX, y, rowSty, players)
		// s.DrawEmpty(playersColX+len(players), y, pingColX-1, y, rowSty)
		// s.DrawText(pingColX, y, rowSty, ping)
		// s.DrawEmpty(pingColX+len(ping), y, tableX2, y, rowSty)

		i++
	}

	// // Draw selected row
	
	// v.mu.RUnlock()
}
