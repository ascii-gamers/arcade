package arcade

import (
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type LobbyCreateView struct {
	View
	selectedRow int	
}
const lcv_borderIndex = 28

var lcv_game_input_default = "[i] to edit, [enter] to save"

var lcv_privateOpt = [2]string {"no", "yes"}
var lcv_gameOpt = [2]string {Tron, Pong}

var lcv_tronPlayerOpt = [7]string {"2","3","4","5","6","7","8"}
var lcv_pongPlayerOpt = [1]string {"2"}
var lcv_playerOpt = [2][]string {lcv_tronPlayerOpt[:], lcv_pongPlayerOpt[:]}

var lcv_game_name = ""
var lcv_game_user_input_indices = [4]int{-1, 0, 0, 0}
var lcv_game_input_categories = [4]string {"NAME", "PRIVATE?", "GAME TYPE", "CAPACITY"}
var lcv_editing = false
var lcv_inputString = ""


const (
	lcv_lobbyTableX1 = 16
	lcv_lobbyTableY1 = 7
	lcv_lobbyTableX2 = 63
	lcv_lobbyTableY2 = 18
)

var create_game_header = []string{
	"| █▀▀ █▀█ █▀▀ ▄▀█ ▀█▀ █▀▀   █▀▀ ▄▀█ █▄█ █▀▀ |",
	"| █▄▄ █▀▄ ██▄ █▀█  █  ██▄   █▄█ █▀█ █ █ ██▄ |",										  
}

var lcv_game_footer = []string{
	"[P]ublish game       [C]ancel",
}

func NewLobbyCreateView() *LobbyCreateView {
	return &LobbyCreateView{}
}

func (v *LobbyCreateView) Init() {
}

func (v *LobbyCreateView) ProcessEvent(evt tcell.Event) {
	switch evt := evt.(type) {
	case *tcell.EventKey:
		switch evt.Key() {
		case tcell.KeyDown:
			v.selectedRow++
			if v.selectedRow > len(lcv_game_input_categories)-1 {
				v.selectedRow = len(lcv_game_input_categories) - 1
			}
			lcv_editing = false
		case tcell.KeyUp:
			v.selectedRow--

			if v.selectedRow < 0 {
				v.selectedRow = 0
			}
			lcv_editing = false
		case tcell.KeyEnter:
			lcv_game_name = lcv_inputString
			lcv_editing = false
			lcv_inputString = ""
		case tcell.KeyLeft:
			lcv_game_user_input_indices[v.selectedRow]--
			if lcv_game_user_input_indices[v.selectedRow] < 0 {
				lcv_game_user_input_indices[v.selectedRow] = 0
			}
			// if game type changes, reset player num
			if v.selectedRow == 2{
				lcv_game_user_input_indices[3] = 0
			}
		case tcell.KeyRight:
			lcv_game_user_input_indices[v.selectedRow]++
			// all other selectors have 2 choices
			maxLength := 2
			if v.selectedRow == 3 {
				// dependent on game type
				maxLength = len(lcv_playerOpt[lcv_game_user_input_indices[v.selectedRow-1]])
			}
			if lcv_game_user_input_indices[v.selectedRow] > maxLength-1 {
				lcv_game_user_input_indices[v.selectedRow] = maxLength - 1
			}
			// if game type changes, reset player num
			if v.selectedRow == 2{
				lcv_game_user_input_indices[3] = 0
			}
		case tcell.KeyRune:
			if !lcv_editing {
				switch evt.Rune() {
				case 'c':
					mgr.SetView(NewGamesListView())
				case 'p':
					// save things
					if lcv_game_name == "" {
						v.selectedRow = 0
						if lcv_game_input_default[0] != '*' {
							lcv_game_input_default = "*" + lcv_game_input_default
						}
					}
					intVar, _ := strconv.Atoi(lcv_playerOpt[lcv_game_user_input_indices[2]][lcv_game_user_input_indices[3]])
					pendingGame = CreatePendingGame(lcv_game_name, (lcv_game_user_input_indices[1] == 1), lcv_gameOpt[lcv_game_user_input_indices[2]], intVar)
					fmt.Println(server.clients)
					hostPlayer := Player{
						Client: *server.clients[server.ID],
						Username: "bob123",
						Host: true,
					}
					pendingGame.AddPlayer(&hostPlayer)
					mgr.SetView(NewLobbyView())
				case 'i':
					lcv_editing = true
					lcv_inputString = ""
				}
			} else {
				lcv_inputString += string(evt.Rune())
			}
			
		}
	}
}

func (v *LobbyCreateView) ProcessPacket(p interface{}) interface{} {
	return nil
}

func (v *LobbyCreateView) Render(s *Screen) {
	width, height := s.Size()

	if lcv_editing {
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
	s.DrawBox(lcv_lobbyTableX1-1, 7, lcv_lobbyTableX2+1, lcv_lobbyTableY2+1, sty_game, true)

	// Draw footer with navigation keystrokes
	s.DrawText((width-len(lcv_game_footer[0]))/2, height-2, sty_game, lcv_game_footer[0])

	// Draw column headers
	
	// s.DrawText(gameColX, 5, sty, "GAME")
	// s.DrawText(playersColX, 5, sty, "PLAYERS")
	// s.DrawText(pingColX, 5, sty, "PING")

	// // Draw border below column headers
	s.DrawLine(lcv_borderIndex, 7, lcv_borderIndex, tableY2, sty_game, true)
	s.DrawText(lcv_borderIndex, 7, sty_game, "╦")
	s.DrawText(lcv_borderIndex, tableY2, sty_game, "╩")

	// Draw selected row
	selectedSty := tcell.StyleDefault.Background(tcell.ColorDarkGreen).Foreground(tcell.ColorWhite)

	i := 0
	for index, inputField := range lcv_game_input_categories {

		y := lcv_lobbyTableY1 + i + 1
		rowSty := sty_game

		if i == v.selectedRow {
			rowSty = selectedSty
		}

		s.DrawEmpty(lcv_lobbyTableX1, y, lcv_lobbyTableX1, y, rowSty)
		s.DrawText(lcv_lobbyTableX1 + 1, y, rowSty, inputField)
		s.DrawEmpty(lcv_lobbyTableX1+len(inputField)+1, y, lcv_borderIndex-1, y, rowSty)

		categoryInputString := lcv_game_name
		categoryIndex := lcv_game_user_input_indices[index]
		thisCategoryMaxLength := 1

		// regarding name
		switch inputField {
		case "NAME":
			if lcv_editing && i == v.selectedRow{
				categoryInputString = lcv_inputString
			} else if lcv_game_name == ""{
				categoryInputString = lcv_game_input_default
			}
		case "PRIVATE?":
			categoryInputString = lcv_privateOpt[categoryIndex]
			thisCategoryMaxLength = len(lcv_privateOpt)
		case "GAME TYPE":
			categoryInputString = lcv_gameOpt[categoryIndex]
			thisCategoryMaxLength = len(lcv_gameOpt)
		case "CAPACITY":
			categoryInputString = lcv_playerOpt[lcv_game_user_input_indices[index-1]][categoryIndex]
			thisCategoryMaxLength = len(lcv_playerOpt[lcv_game_user_input_indices[index-1]])
		}

		if categoryIndex != -1 {
			if categoryIndex < thisCategoryMaxLength - 1 {
				categoryInputString += " →"
			}
			if categoryIndex > 0 {
				categoryInputString = "← " + categoryInputString
			}
		}

		categoryX := (lcv_lobbyTableX2 - lcv_borderIndex - utf8.RuneCountInString(categoryInputString)) / 2 + lcv_borderIndex
		s.DrawEmpty(lcv_borderIndex+1, y, categoryX-1, y, rowSty)
		s.DrawText(categoryX, y, rowSty, categoryInputString)
		s.DrawEmpty(categoryX+utf8.RuneCountInString(categoryInputString), y, lcv_lobbyTableX2-1, y, rowSty)

		i++
	}

	// // Draw selected row
	
	// v.mu.RUnlock()
}
