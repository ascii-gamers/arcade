package arcade

import (
	"github.com/gdamore/tcell/v2"
)

type LobbyView struct {
	View
}

func NewLobbyView() *LobbyView {
	return &LobbyView{}
}

func (v *LobbyView) Init() {
}

func (v *LobbyView) ProcessEvent(evt tcell.Event) {
	// TODO
}

func (v *LobbyView) ProcessPacket(p interface{}) interface{} {
	switch p.(type) {
	case HelloMessage:
		return NewLobbyInfoMessage(server.Addr)
	}

	return nil
}

func (v *LobbyView) Render(s *Screen) {
	s.DrawText(0, 0, tcell.StyleDefault, "hi soomin")
}
