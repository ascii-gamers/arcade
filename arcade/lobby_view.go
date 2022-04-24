package arcade

import (
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type LobbyView struct {
	View
	selectedRow int	
}
var borderIndex = 28

const game_input_default = "[i] to edit, [enter] to save"

var privateOpt = [2]string {"no", "yes"}
var gameOpt = [2]string {"tron", "pong"}

var tronPlayerOpt = [7]string {"2","3","4","5","6","7","8"}
var pongPlayerOpt = [1]string {"2"}
var playerOpt = [2][]string {tronPlayerOpt[:], pongPlayerOpt[:]}

var game_user_input = ""
var game_user_input_indices = [4]int{-1, 0, 0, 0}
var game_input_categories = [4]string {"NAME", "PRIVATE?", "GAME TYPE", "CAPACITY"}
var editing = false
var inputString = ""


const (
	lobbyTableX1 = 16
	lobbyTableY1 = 7
	lobbyTableX2 = 63
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
			if v.selectedRow > len(game_input_categories)-1 {
				v.selectedRow = len(game_input_categories) - 1
			}
			editing = false
		case tcell.KeyUp:
			v.selectedRow--

			if v.selectedRow < 0 {
				v.selectedRow = 0
			}
			editing = false
		case tcell.KeyEnter:
			game_user_input = inputString
			editing = false
			inputString = ""
		case tcell.KeyLeft:
			game_user_input_indices[v.selectedRow]--
			if game_user_input_indices[v.selectedRow] < 0 {
				game_user_input_indices[v.selectedRow] = 0
			}
			// if game type changes, reset player num
			if v.selectedRow == 2{
				game_user_input_indices[3] = 0
			}
		case tcell.KeyRight:
			game_user_input_indices[v.selectedRow]++
			// all other selectors have 2 choices
			maxLength := 2
			if v.selectedRow == 3 {
				// dependent on game type
				maxLength = len(playerOpt[game_user_input_indices[v.selectedRow-1]])
			}
			if game_user_input_indices[v.selectedRow] > maxLength-1 {
				game_user_input_indices[v.selectedRow] = maxLength - 1
			}
			// if game type changes, reset player num
			if v.selectedRow == 2{
				game_user_input_indices[3] = 0
			}
		case tcell.KeyRune:
			if !editing {
				switch evt.Rune() {
				case 'c':
					mgr.SetView(NewGamesListView())
				case 'p':
					// save things
					mgr.SetView(NewLobbyView())
				case 'i':
					editing = true
					inputString = ""
				}
			} else {
				inputString += string(evt.Rune())
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

	if editing {
		s.SetCursorStyle(tcell.CursorStyleBlinkingBlock)
	} else {
		s.SetCursorStyle(tcell.CursorStyleDefault)
	}

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
	s.DrawLine(borderIndex, 7, borderIndex, tableY2, sty_game, true)
	s.DrawText(borderIndex, 7, sty_game, "╦")
	s.DrawText(borderIndex, tableY2, sty_game, "╩")

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
		s.DrawEmpty(lobbyTableX1+len(inputField)+1, y, borderIndex-1, y, rowSty)

		categoryInputString := game_user_input
		categoryIndex := game_user_input_indices[index]

		// regarding name
		switch inputField {
		case "NAME":
			if editing && i == v.selectedRow{
				categoryInputString = inputString
			} else if game_user_input == ""{
				categoryInputString = game_input_default
			}
		case "PRIVATE?":
			categoryInputString = privateOpt[categoryIndex]
		case "GAME TYPE":
			categoryInputString = gameOpt[categoryIndex]
		case "CAPACITY":
			categoryInputString = playerOpt[game_user_input_indices[index-1]][categoryIndex]
		}

		// if categoryIndex != -1 {
		// 	if categoryIndex == 0 {

		// 	}
		// }

		categoryX := (lobbyTableX2 - borderIndex - utf8.RuneCountInString(categoryInputString)) / 2 + borderIndex
		s.DrawEmpty(borderIndex+1, y, categoryX-1, y, rowSty)
		s.DrawText(categoryX, y, rowSty, categoryInputString)
		s.DrawEmpty(categoryX+len(categoryInputString), y, lobbyTableX2-1, y, rowSty)

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
