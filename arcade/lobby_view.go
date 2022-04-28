package arcade

import (
	"fmt"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type LobbyView struct {
	View
	selectedRow int	
}

const (
	lv_TableX1 = 20
	lv_TableY1 = 4
	lv_TableX2 = 59
	lv_TableY2 = 12
)

var lobby_footer = []string{
	"[S]tart game       [C]ancel",
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
		case tcell.KeyRune:
			switch evt.Rune() {
			case 'c':
				mgr.SetView(NewGamesListView())
			case 's':
				//start game
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
	sty := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorGreen)
	sty_bold := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorDarkGreen)

	// Draw GAME header

	game_header := pong_header
	if game.GameType == "tron" {
		game_header = tron_header
	}  
	headerX := (width - utf8.RuneCountInString(game_header[0])) / 2
	s.DrawText(headerX, 1, sty, game_header[0])
	s.DrawText(headerX, 2, sty, game_header[1])

	// Draw box surrounding games list
	s.DrawBox(lv_TableX1, lv_TableY1, lv_TableX2, lv_TableY2, sty, true)

	// Draw game info 

	// name
	nameHeader := "Name: "
	nameString := game.Name
	s.DrawText((width-len(nameHeader + nameString))/2, lv_TableY1+1, sty, nameHeader)
	s.DrawText((width-len(nameHeader + nameString))/2+utf8.RuneCountInString(nameHeader), lv_TableY1+1, sty_bold, nameString)

	// private
	privateHeader := "Visibility: "
	privateString := "public"
	if game.Private {
		privateString = "private, Code: " + game.Code
	} 
	s.DrawText((width-len(privateHeader + privateString))/2, lv_TableY1+2, sty, privateHeader)
	s.DrawText((width-len(privateHeader + privateString))/2+utf8.RuneCountInString(privateHeader), lv_TableY1+2, sty_bold, privateString)

	// capacity
	capacityHeader := "Game capacity: "
	capacityString := fmt.Sprintf("(%v/%v)", game.NumFull, game.Capacity)
	s.DrawText((width-len(capacityHeader + capacityString))/2, lv_TableY1+3, sty, capacityHeader)
	s.DrawText((width-len(capacityHeader + capacityString))/2+utf8.RuneCountInString(capacityHeader), lv_TableY1+3, sty_bold, capacityString)

	// // Draw footer with navigation keystrokes
	s.DrawText((width-len(lobby_footer[0]))/2, height-2, sty, lobby_footer[0])

	// // Draw column headers
	
	// // Draw selected row
	// selectedSty := tcell.StyleDefault.Background(tcell.ColorDarkGreen).Foreground(tcell.ColorWhite)

	// i := 0
	// for index, inputField := range game_input_categories {

	// 	y := lobbyTableY1 + i + 1
	// 	rowSty := sty

	// 	if i == v.selectedRow {
	// 		rowSty = selectedSty
	// 	}

	// 	s.DrawEmpty(lobbyTableX1, y, lobbyTableX1, y, rowSty)
	// 	s.DrawText(lobbyTableX1 + 1, y, rowSty, inputField)
	// 	s.DrawEmpty(lobbyTableX1+len(inputField)+1, y, borderIndex-1, y, rowSty)

	// 	categoryInputString := game_name
	// 	categoryIndex := game_user_input_indices[index]
	// 	thisCategoryMaxLength := 1

	// 	// regarding name
	// 	switch inputField {
	// 	case "NAME":
	// 		if editing && i == v.selectedRow{
	// 			categoryInputString = inputString
	// 		} else if game_name == ""{
	// 			categoryInputString = game_input_default
	// 		}
	// 	case "PRIVATE?":
	// 		categoryInputString = privateOpt[categoryIndex]
	// 		thisCategoryMaxLength = len(privateOpt)
	// 	case "GAME TYPE":
	// 		categoryInputString = gameOpt[categoryIndex]
	// 		thisCategoryMaxLength = len(gameOpt)
	// 	case "CAPACITY":
	// 		categoryInputString = playerOpt[game_user_input_indices[index-1]][categoryIndex]
	// 		thisCategoryMaxLength = len(playerOpt[game_user_input_indices[index-1]])
	// 	}

	// 	if categoryIndex != -1 {
	// 		if categoryIndex < thisCategoryMaxLength - 1 {
	// 			categoryInputString += " →"
	// 		}
	// 		if categoryIndex > 0 {
	// 			categoryInputString = "← " + categoryInputString
	// 		}
	// 	}

	// 	categoryX := (lobbyTableX2 - borderIndex - utf8.RuneCountInString(categoryInputString)) / 2 + borderIndex
	// 	s.DrawEmpty(borderIndex+1, y, categoryX-1, y, rowSty)
	// 	s.DrawText(categoryX, y, rowSty, categoryInputString)
	// 	s.DrawEmpty(categoryX+utf8.RuneCountInString(categoryInputString), y, lobbyTableX2-1, y, rowSty)

	// 	i++
	// }

	// // Draw selected row
	
	// v.mu.RUnlock()
}
