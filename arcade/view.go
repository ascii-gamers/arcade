package arcade

import (
	"github.com/gdamore/tcell/v2"
)

type View interface {
	Init()
	ProcessEvent(ev tcell.Event)
	ProcessMessage(from *Client, p interface{}) interface{}
	Render(s *Screen)
}
