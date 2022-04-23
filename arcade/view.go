package arcade

import (
	"github.com/gdamore/tcell/v2"
)

type View interface {
	Init()
	ProcessEvent(ev tcell.Event)
	ProcessPacket(p interface{}) interface{}
	Render(s *Screen)
}
