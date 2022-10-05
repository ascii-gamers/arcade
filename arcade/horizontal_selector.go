package arcade

import (
	"strconv"

	"github.com/gdamore/tcell/v2"
)

type HorizontalSelectorOptions[T int | string] struct {
	LayoutOptions

	Border       bool
	Label        string
	LabelPadding int
	Values       []T
}

type HorizontalSelector[T int | string] struct {
	BaseComponent

	sty    tcell.Style
	index  int
	active bool
	HorizontalSelectorOptions[T]
}

func NewHorizontalSelector[T int | string](opts HorizontalSelectorOptions[T]) *HorizontalSelector[T] {
	return &HorizontalSelector[T]{
		sty:                       tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen),
		HorizontalSelectorOptions: opts,
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
			if s.index >= len(s.Values)-1 {
				break
			}

			s.index += 1
		}
	}
}

func (sel *HorizontalSelector[T]) Render(s *Screen) {
	sel.RLock()
	defer sel.RUnlock()

	// screenW, screenH := s.displaySize()

	x := sel.X
	y := sel.Y

	// switch x {
	// case CenterX:
	// 	x = (screenW - sel.Width) / 2
	// }

	// switch y {
	// case CenterY:
	// 	y = (screenH - 2) / 2
	// }

	if sel.Border {
		s.DrawBox(x, y, x+sel.ContentWidth-1, y+2, sel.sty, false)
		s.DrawText(x+(sel.ContentWidth-len(sel.Label))/2, y-1, sel.sty, sel.Label)
	} else {
		s.DrawText(x-sel.LabelPadding-len(sel.Label), y, sel.sty, sel.Label)
	}

	text := ""

	if len(sel.Values) > 0 {
		switch value := any(sel.Values[sel.index]).(type) {
		case int:
			text = strconv.Itoa(value)
		case string:
			text = value
		}
	}

	s.DrawText(x+(sel.ContentWidth-len(text))/2, y+1, sel.sty, text)

	if sel.index > 0 {
		s.DrawText(x-2, y+1, sel.sty, "◄")
	}

	if sel.index < len(sel.Values)-1 {
		s.DrawText(x+sel.ContentWidth+1, y+1, sel.sty, "►")
	}
}
