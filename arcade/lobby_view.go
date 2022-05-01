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

// const stickmen = []string{
// 	o   \ o /  _ o         __|    \ /     |__        o _  \ o /   o
// 	/|\    |     /\   ___\o   \o    |    o/    o/__   /\     |    /|\
// 	/ \   / \   | \  /)  |    ( \  /o\  / )    |  (\  / |   / \   / \
// }

// const stickmen_list = [][]string{{" o ","/|\\","/ \\"}, {"\\ o /","  |  "," / \\ "}, }

// var simple_man = []string {" o ","/|\\","/ \\"};

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

func (v *LobbyView) ProcessEvent(evt interface{}) {
	switch evt := evt.(type) {
	case *ClientDisconnectEvent:
		lobby.RemovePlayer(evt.ClientID)
	case *tcell.EventKey:
		switch evt.Key() {
		case tcell.KeyRune:
			switch evt.Rune() {
			case 'c':
				if lobby.HostID != server.ID {
					// not the host, just leave the game
					host, _ := server.GetClient(lobby.HostID)
					go host.Send(NewLeaveMessage(&Player{}))
				}
				mgr.SetView(NewGamesListView())
				// delete game?
			case 's':
				//start game
				NewGame(lobby)
			}
		}
	}
}

func (v *LobbyView) ProcessMessage(from *Client, p interface{}) interface{} {
	switch p := p.(type) {
	case HelloMessage:
		return NewLobbyInfoMessage(lobby)
	case JoinMessage:
		lobby.Lock()
		if len(lobby.PlayerIDs) == lobby.Capacity {
			lobby.Unlock()
			return NewJoinReplyMessage(&Lobby{}, ErrCapacity)
		} else if lobby.code != p.Code {
			lobby.Unlock()
			return NewJoinReplyMessage(&Lobby{}, ErrWrongCode)
		} else {
			lobby.Unlock()
			lobby.AddPlayer(p.Player.ClientID)
			return NewJoinReplyMessage(lobby, OK)
		}
	case LeaveMessage:
		lobby.RemovePlayer(p.Player.ClientID)
	}
	return nil
}

func (v *LobbyView) Render(s *Screen) {
	width, height := s.displaySize()

	// Green text on default background
	sty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen)
	sty_bold := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorDarkGreen)

	// Draw GAME header

	game_header := pong_header
	if lobby.GameType == Tron {
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
	nameString := lobby.Name
	s.DrawText((width-len(nameHeader+nameString))/2, lv_TableY1+1, sty, nameHeader)
	s.DrawText((width-len(nameHeader+nameString))/2+utf8.RuneCountInString(nameHeader), lv_TableY1+1, sty_bold, nameString)

	// private
	privateHeader := "Visibility: "
	privateString := "public"
	if lobby.Private {
		privateString = "private, Join Code: " + lobby.code
	}
	s.DrawText((width-len(privateHeader+privateString))/2, lv_TableY1+2, sty, privateHeader)
	s.DrawText((width-len(privateHeader+privateString))/2+utf8.RuneCountInString(privateHeader), lv_TableY1+2, sty_bold, privateString)

	// capacity
	capacityHeader := "Game capacity: "
	capacityString := fmt.Sprintf("(%v/%v)", len(lobby.PlayerIDs), lobby.Capacity)
	s.DrawText((width-len(capacityHeader+capacityString))/2, lv_TableY1+3, sty, capacityHeader)
	s.DrawText((width-len(capacityHeader+capacityString))/2+utf8.RuneCountInString(capacityHeader), lv_TableY1+3, sty_bold, capacityString)

	// Draw people
	s.DrawText((width-len(capacityHeader+capacityString))/2+utf8.RuneCountInString(capacityHeader), lv_TableY1+3, sty_bold, capacityString)

	// Draw footer with navigation keystrokes
	s.DrawText((width-len(lobby_footer[0]))/2, height-2, sty, lobby_footer[0])

}

func (v *LobbyView) Unload() {
}
