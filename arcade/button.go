package arcade

import (
	"github.com/gdamore/tcell/v2"
)

const BUTTON_HEIGHT = 3

type Button struct {
	BaseComponent

	x, y, width int
	title       string
	active      bool
	action      func()
}

func NewButton(x, y, width int, title string, action func()) *Button {
	return &Button{
		x:      x,
		y:      y,
		width:  width,
		title:  title,
		action: action,
	}
}

func (b *Button) Focus() {
	b.Lock()
	defer b.Unlock()

	b.active = true
}

func (b *Button) ProcessEvent(evt interface{}) {
	b.Lock()
	defer b.Unlock()

	switch evt := evt.(type) {
	case *tcell.EventKey:
		switch evt.Key() {
		case tcell.KeyDown, tcell.KeyTab:
			if b.delegate.NavigateForward() {
				b.active = false
			}
		case tcell.KeyUp:
			if b.delegate.NavigateBackward() {
				b.active = false
			}
		case tcell.KeyEnter:
			b.action()
		}
	}
}

func (b *Button) Render(s *Screen) {
	b.RLock()
	defer b.RUnlock()

	screenW, screenH := s.displaySize()

	x := b.x
	y := b.y

	switch x {
	case CenterX:
		x = (screenW - b.width) / 2
	}

	switch y {
	case CenterY:
		y = (screenH - BUTTON_HEIGHT) / 2
	}

	color := tcell.ColorGreen

	if b.active {
		color = tcell.ColorTeal
	}

	s.DrawEmpty(x, y, x+b.width-1, y+BUTTON_HEIGHT-1, tcell.StyleDefault.Background(color))
	s.DrawText(x+(b.width-len(b.title))/2, y+1, tcell.StyleDefault.Background(color).Foreground(tcell.ColorBlack), b.title)
}
