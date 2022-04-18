package arcade

import (
	"github.com/gdamore/tcell/v2"
)

type View interface {
	ProcessEvent(ev tcell.Event)
	Render(s *Screen)
}
