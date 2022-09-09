package arcade

import (
	"arcade/arcade/net"
	"encoding"
	"encoding/json"
	"fmt"
	"sync"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type LobbyView struct {
	View
	mgr *ViewManager

	sync.RWMutex
	Lobby *Lobby
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

func NewLobbyView(mgr *ViewManager, lobby *Lobby) *LobbyView {
	return &LobbyView{
		mgr:   mgr,
		Lobby: lobby,
	}
}

func (v *LobbyView) Init() {
}

func (v *LobbyView) ProcessEvent(evt interface{}) {
	switch evt := evt.(type) {
	case *ClientDisconnectedEvent:
		if v.Lobby.HostID == arcade.Server.ID {
			v.Lobby.RemovePlayer(evt.ClientID)
		}
	case *HeartbeatEvent:
		if v.Lobby.HostID != arcade.Server.ID {
			lobby := new(Lobby)
			json.Unmarshal(evt.Metadata, lobby)
			// fmt.Println("lobby updated w heartbeat")
			v.Lock()
			v.Lobby = lobby
			v.Unlock()
		}
		// do something with lobby
	case *tcell.EventKey:
		switch evt.Key() {
		case tcell.KeyRune:
			switch evt.Rune() {
			case 'c':
				v.Lobby.mu.RLock()
				if v.Lobby.HostID != arcade.Server.ID {
					// not the host, just leave the game
					host, _ := arcade.Server.Network.GetClient(v.Lobby.HostID)
					v.Lobby.mu.RUnlock()

					arcade.Server.Network.Send(host, NewLeaveMessage(arcade.Server.ID, v.Lobby.ID))

					arcade.Server.EndAllHeartbeats()
					v.mgr.SetView(NewGamesListView(v.mgr))
				} else {
					// first extract lobbyID for messages
					lobbyID := v.Lobby.ID
					v.Lobby.mu.RUnlock()

					arcade.Server.EndAllHeartbeats()
					// send updates to everyone

					arcade.Server.Network.ClientsRange(func(client *net.Client) bool {
						if client.Distributor {
							return true
						}

						arcade.Server.Network.Send(client, NewLobbyEndMessage(lobbyID))

						return true
					})

					v.mgr.SetView(NewGamesListView(v.mgr))

				}
			case 's':
				//start gamex
				v.Lobby.mu.RLock()
				if v.Lobby.HostID == arcade.Server.ID {
					for _, playerId := range v.Lobby.PlayerIDs {
						client, ok := arcade.Server.Network.GetClient(playerId)
						if ok {
							arcade.Server.Network.Send(client, NewStartGameMessage(v.Lobby.ID))
						}
					}
					NewGame(v.mgr, v.Lobby)
				}
				v.Lobby.mu.RUnlock()
			}
		}
	}
}

func (v *LobbyView) ProcessMessage(from *net.Client, p interface{}) interface{} {
	switch p := p.(type) {
	case *HelloMessage:
		return NewLobbyInfoMessage(v.Lobby)
	case *JoinMessage:
		if v.Lobby.HostID == arcade.Server.ID {
			if v.Lobby.ID == p.LobbyID {
				v.Lobby.mu.RLock()
				playerIDlength := len(v.Lobby.PlayerIDs)
				cap := v.Lobby.Capacity
				lobby_code := v.Lobby.Code
				v.Lobby.mu.RUnlock()

				if playerIDlength == cap {
					return NewJoinReplyMessage(&Lobby{}, ErrCapacity)
				} else if lobby_code != p.Code {
					return NewJoinReplyMessage(&Lobby{}, ErrWrongCode)
				} else {
					v.Lobby.AddPlayer(p.PlayerID)
					arcade.Server.BeginHeartbeats(p.PlayerID)
					return NewJoinReplyMessage(v.Lobby, OK)
				}
			} else {
				// send lobby end
				return NewLobbyEndMessage(v.Lobby.ID)
			}
		}

	case *LeaveMessage:
		if v.Lobby.ID == p.LobbyID && v.Lobby.HostID == arcade.Server.ID {
			v.Lobby.RemovePlayer(p.PlayerID)
		}

		arcade.Server.EndHeartbeats(p.PlayerID)
		v.mgr.RequestRender()
	case *LobbyEndMessage:
		// get rid of lobby
		if v.Lobby.ID == p.LobbyID {
			v.Lobby = &Lobby{}

			arcade.Server.EndAllHeartbeats()
			v.mgr.SetView(NewGamesListView(v.mgr))
		}
	case *StartGameMessage:
		if p.GameID == v.Lobby.ID {
			NewGame(v.mgr, v.Lobby)
		}

		return nil
	}

	return nil
}

func (v *LobbyView) Render(s *Screen) {
	v.Lobby.mu.Lock()
	defer v.Lobby.mu.Unlock()

	width, height := s.displaySize()

	// Green text on default background
	sty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen)
	sty_bold := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorDarkGreen)

	// Draw GAME header
	s.DrawBlockText(CenterX, 1, sty, "TRON", false)

	// Draw box surrounding games list
	s.DrawBox(lv_TableX1, lv_TableY1, lv_TableX2, lv_TableY2, sty, true)

	// Draw game info

	// name
	nameHeader := "Name: "
	nameString := v.Lobby.Name
	s.DrawText((width-len(nameHeader+nameString))/2, lv_TableY1+1, sty, nameHeader)
	s.DrawText((width-len(nameHeader+nameString))/2+utf8.RuneCountInString(nameHeader), lv_TableY1+1, sty_bold, nameString)

	// private
	privateHeader := "Visibility: "
	privateString := "public"
	if v.Lobby.Private {
		privateString = "private, Join Code: " + v.Lobby.Code
	}
	s.DrawText((width-len(privateHeader+privateString))/2, lv_TableY1+2, sty, privateHeader)
	s.DrawText((width-len(privateHeader+privateString))/2+utf8.RuneCountInString(privateHeader), lv_TableY1+2, sty_bold, privateString)

	// capacity
	capacityHeader := "Game capacity: "
	capacityString := fmt.Sprintf("(%v/%v)", len(v.Lobby.PlayerIDs), v.Lobby.Capacity)
	s.DrawText((width-len(capacityHeader+capacityString))/2, lv_TableY1+3, sty, capacityHeader)
	s.DrawText((width-len(capacityHeader+capacityString))/2+utf8.RuneCountInString(capacityHeader), lv_TableY1+3, sty_bold, capacityString)

	// Draw footer with navigation keystrokes
	if arcade.Server.ID == v.Lobby.HostID {
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
	if v.Lobby.HostID == arcade.Server.ID {
		// send to all the players, similar to 'c'
		lobbyID := v.Lobby.ID

		arcade.Server.Network.ClientsRange(func(client *net.Client) bool {
			if client.Distributor {
				return true
			}

			arcade.Server.Network.Send(client, NewLobbyEndMessage(lobbyID))

			return true
		})
	} else {
		// only send to host
		host, ok := arcade.Server.Network.GetClient(v.Lobby.HostID)

		if ok {
			arcade.Server.Network.Send(host, NewLeaveMessage(arcade.Server.ID, v.Lobby.ID))
		}
	}
}

func (v *LobbyView) GetHeartbeatMetadata() encoding.BinaryMarshaler {
	v.RLock()
	defer v.RUnlock()

	return v.Lobby
}
