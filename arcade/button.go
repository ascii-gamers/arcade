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
	b := &Button{
		x:      x,
		y:      y,
		width:  width,
		title:  title,
		action: action,
	}

	if x >= CenterXPlaceholder {
		b.x = (x - CenterXPlaceholder - width) / 2
	}

	if y >= CenterYPlaceholder {
		b.y = (y - CenterYPlaceholder - BUTTON_HEIGHT) / 2
	}

	return b
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

	color := tcell.ColorGreen

	if b.active {
		color = tcell.ColorTeal
	}

	s.DrawEmpty(b.x, b.y, b.x+b.width-1, b.y+BUTTON_HEIGHT-1, tcell.StyleDefault.Background(color))
	s.DrawText(b.x+(b.width-len(b.title))/2, b.y+1, tcell.StyleDefault.Background(color).Foreground(tcell.ColorBlack), b.title)
}
