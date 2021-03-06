package arcade

import (
	"encoding"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type LobbyView struct {
	View
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

var lobby_footer_host = []string{
	"[S]tart game       [C]ancel",
}

var lobby_footer_nonhost = []string{
	"[C]ancel",
}

func NewLobbyView() *LobbyView {
	return &LobbyView{}
}

func (v *LobbyView) Init() {
}

func (v *LobbyView) ProcessEvent(evt interface{}) {
	arcade.lobbyMux.Lock()
	defer arcade.lobbyMux.Unlock()

	switch evt := evt.(type) {
	case *ClientDisconnectEvent:
		if arcade.Lobby.HostID == arcade.Server.ID {
			arcade.Lobby.RemovePlayer(evt.ClientID)
		}
	case *HeartbeatEvent:
		if arcade.Lobby.HostID != arcade.Server.ID {
			lobby := new(Lobby)
			json.Unmarshal(evt.Metadata, lobby)
			// fmt.Println("lobby updated w heartbeat")
			arcade.Lobby = lobby
		}
		// do something with lobby
	case *tcell.EventKey:
		switch evt.Key() {
		case tcell.KeyRune:
			switch evt.Rune() {
			case 'c':
				arcade.Lobby.mu.RLock()
				if arcade.Lobby.HostID != arcade.Server.ID {
					// not the host, just leave the game
					host, _ := arcade.Server.Network.GetClient(arcade.Lobby.HostID)
					arcade.Lobby.mu.RUnlock()

					arcade.Server.Network.Send(host, NewLeaveMessage(arcade.Server.ID, arcade.Lobby.ID))

					arcade.Lobby = &Lobby{}

					arcade.Server.EndAllHeartbeats()
					arcade.ViewManager.SetView(NewGamesListView())
				} else {
					// first extract lobbyID for messages
					lobbyID := arcade.Lobby.ID
					arcade.Lobby.mu.RUnlock()

					// get rid of lobby
					arcade.Lobby = &Lobby{}

					arcade.Server.EndAllHeartbeats()
					// send updates to everyone

					arcade.Server.Network.ClientsRange(func(client *Client) bool {
						if client.Distributor {
							return true
						}

						arcade.Server.Network.Send(client, NewLobbyEndMessage(lobbyID))

						return true
					})

					arcade.ViewManager.SetView(NewGamesListView())

				}
			case 's':
				//start gamex
				arcade.Lobby.mu.RLock()
				if arcade.Lobby.HostID == arcade.Server.ID {
					for _, playerId := range arcade.Lobby.PlayerIDs {
						client, ok := arcade.Server.Network.GetClient(playerId)
						if ok {
							arcade.Server.Network.Send(client, NewStartGameMessage(arcade.Lobby.ID))
						}
					}
					NewGame(arcade.Lobby)
				}
				arcade.Lobby.mu.RUnlock()
			}
		}
	}
}

func (v *LobbyView) ProcessMessage(from *Client, p interface{}) interface{} {
	arcade.lobbyMux.Lock()
	defer arcade.lobbyMux.Unlock()

	arcade.Lobby.mu.RLock()
	lobbyID := arcade.Lobby.ID
	arcade.Lobby.mu.RUnlock()

	switch p := p.(type) {
	case HelloMessage:
		// return nil
		return NewLobbyInfoMessage(arcade.Lobby)
	case JoinMessage:
		if arcade.Lobby.HostID == arcade.Server.ID {
			if lobbyID == p.LobbyID {
				arcade.Lobby.mu.RLock()
				playerIDlength := len(arcade.Lobby.PlayerIDs)
				cap := arcade.Lobby.Capacity
				lobby_code := arcade.Lobby.Code
				arcade.Lobby.mu.RUnlock()

				if playerIDlength == cap {
					return NewJoinReplyMessage(&Lobby{}, ErrCapacity)
				} else if lobby_code != p.Code {
					return NewJoinReplyMessage(&Lobby{}, ErrWrongCode)
				} else {
					arcade.Lobby.AddPlayer(p.PlayerID)
					arcade.Server.BeginHeartbeats(p.PlayerID)
					return NewJoinReplyMessage(arcade.Lobby, OK)
				}
			} else {
				// send lobby end
				return NewLobbyEndMessage(lobbyID)
			}
		}

	case LeaveMessage:
		// panic("0")
		if lobbyID == p.LobbyID && arcade.Lobby.HostID == arcade.Server.ID {
			arcade.Lobby.RemovePlayer(p.PlayerID)
		}
		arcade.Server.EndHeartbeats(p.PlayerID)
		// panic("1")
		arcade.lobbyMux.Unlock()
		arcade.ViewManager.RequestRender()
		arcade.lobbyMux.Lock()
		// panic("2")
	case LobbyEndMessage:
		// get rid of lobby
		if lobbyID == p.LobbyID {
			arcade.Lobby = &Lobby{}

			arcade.Server.EndAllHeartbeats()
			arcade.ViewManager.SetView(NewGamesListView())
		}
	case StartGameMessage:
		if p.GameID == lobbyID {
			NewGame(arcade.Lobby)
		}
		return nil
	}
	return nil
}

func (v *LobbyView) Render(s *Screen) {
	// panic("RENDER PANIC")
	arcade.lobbyMux.RLock()
	defer arcade.lobbyMux.RUnlock()
	arcade.Lobby.mu.Lock()
	defer arcade.Lobby.mu.Unlock()

	width, height := s.displaySize()

	// Green text on default background
	sty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen)
	sty_bold := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorDarkGreen)

	// Draw GAME header

	game_header := pong_header
	if arcade.Lobby.GameType == Tron {
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
	nameString := arcade.Lobby.Name
	s.DrawText((width-len(nameHeader+nameString))/2, lv_TableY1+1, sty, nameHeader)
	s.DrawText((width-len(nameHeader+nameString))/2+utf8.RuneCountInString(nameHeader), lv_TableY1+1, sty_bold, nameString)

	// private
	privateHeader := "Visibility: "
	privateString := "public"
	if arcade.Lobby.Private {
		privateString = "private, Join Code: " + arcade.Lobby.Code
	}
	s.DrawText((width-len(privateHeader+privateString))/2, lv_TableY1+2, sty, privateHeader)
	s.DrawText((width-len(privateHeader+privateString))/2+utf8.RuneCountInString(privateHeader), lv_TableY1+2, sty_bold, privateString)

	// capacity
	capacityHeader := "Game capacity: "
	capacityString := fmt.Sprintf("(%v/%v)", len(arcade.Lobby.PlayerIDs), arcade.Lobby.Capacity)
	s.DrawText((width-len(capacityHeader+capacityString))/2, lv_TableY1+3, sty, capacityHeader)
	s.DrawText((width-len(capacityHeader+capacityString))/2+utf8.RuneCountInString(capacityHeader), lv_TableY1+3, sty_bold, capacityString)

	// Draw footer with navigation keystrokes
	if arcade.Server.ID == arcade.Lobby.HostID {
		// I am host so I should see start game controls
		hostLabelString := "You are the host."
		s.DrawText((width-len(hostLabelString))/2, lv_TableY1+5, sty, hostLabelString)
		s.DrawText((width-len(lobby_footer_host[0]))/2, height-2, sty, lobby_footer_host[0])
	} else {
		participantLabelString := "Waiting for host to start game..."
		s.DrawText((width-len(participantLabelString))/2, lv_TableY1+5, sty, participantLabelString)
		s.DrawText((width-len(lobby_footer_nonhost[0]))/2, height-2, sty, lobby_footer_nonhost[0])
	}

}

func (v *LobbyView) Unload() {
}

func (v *LobbyView) GetHeartbeatMetadata() encoding.BinaryMarshaler {
	arcade.lobbyMux.Lock()
	defer arcade.lobbyMux.Unlock()
	return arcade.Lobby
}
