package arcade

import (
	"strconv"

	"github.com/gdamore/tcell/v2"
)

type HorizontalSelector[T int | string] struct {
	BaseComponent

	sty         tcell.Style
	x, y, width int
	index       int
	values      []T
	label       string
	active      bool
}

func NewHorizontalSelector[T int | string](x, y, width int, label string, values []T) *HorizontalSelector[T] {
	return &HorizontalSelector[T]{
		sty:    tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen),
		x:      x,
		y:      y,
		width:  width,
		label:  label,
		values: values,
	}
}

func (s *HorizontalSelector[T]) Focus() {
	s.Lock()
	defer s.Unlock()

	s.active = true
}

func (s *HorizontalSelector[T]) ProcessEvent(evt interface{}) {
	s.Lock()
	defer s.Unlock()

	switch evt := evt.(type) {
	case *tcell.EventKey:
		switch evt.Key() {
		case tcell.KeyDown, tcell.KeyTab, tcell.KeyEnter:
			if s.delegate.NavigateForward() {
				s.active = false
			}
		case tcell.KeyUp:
			if s.delegate.NavigateBackward() {
				s.active = false
			}
		case tcell.KeyLeft:
			if s.index == 0 {
				break
			}

			s.index -= 1
		case tcell.KeyRight:
			if s.index >= len(s.values)-1 {
				break
			}

			s.index += 1
		}
	}
}

func (sel *HorizontalSelector[T]) Render(s *Screen) {
	sel.RLock()
	defer sel.RUnlock()

	screenW, screenH := s.displaySize()

	x := sel.x
	y := sel.y

	switch x {
	case CenterX:
		x = (screenW - sel.width) / 2
	}

	switch y {
	case CenterY:
		y = (screenH - 2) / 2
	}

	text := ""

	if len(sel.values) > 0 {
		switch value := any(sel.values[sel.index]).(type) {
		case int:
			text = strconv.Itoa(value)
		case string:
			text = value
		}
	}

	s.DrawText(x+(sel.width-len(sel.label))/2, y-1, sel.sty, sel.label)
	s.DrawBox(x, y, x+sel.width-1, y+2, sel.sty, false)
	s.DrawText(x+(sel.width-len(text))/2, y+1, sel.sty, text)

	if sel.index > 0 {
		s.DrawText(x-2, y+1, sel.sty, "◄")
	}

	if sel.index < len(sel.values)-1 {
		s.DrawText(x+sel.width+1, y+1, sel.sty, "►")
	}
}
