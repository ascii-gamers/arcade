package arcade

import (
	"github.com/gdamore/tcell/v2"
)

type TextField struct {
	BaseComponent

	sty         tcell.Style
	x, y, width int
	cursorPos   int
	value       string
	label       string
	active      bool
}

func NewTextField(x, y, width int, label string) *TextField {
	return &TextField{
		sty:   tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen),
		x:     x,
		y:     y,
		width: width,
		label: label,
	}
}

func (tf *TextField) Focus() {
	tf.Lock()
	defer tf.Unlock()

	tf.active = true
}

func (tf *TextField) ProcessEvent(evt interface{}) {
	tf.Lock()
	defer tf.Unlock()

	switch evt := evt.(type) {
	case *tcell.EventKey:
		switch evt.Key() {
		case tcell.KeyDown, tcell.KeyTab, tcell.KeyEnter:
			if tf.delegate.NavigateForward() {
				tf.active = false
			}
		case tcell.KeyUp:
			if tf.delegate.NavigateBackward() {
				tf.active = false
			}
		case tcell.KeyLeft:
			if tf.cursorPos == 0 {
				break
			}

			tf.cursorPos -= 1
		case tcell.KeyRight:
			if tf.cursorPos >= len(tf.value) {
				break
			}

			tf.cursorPos += 1
		case tcell.KeyDelete, tcell.KeyDEL:
			if tf.cursorPos == 0 {
				break
			}

			tf.value = tf.value[:tf.cursorPos-1] + tf.value[tf.cursorPos:]
			tf.cursorPos -= 1
		default:
			tf.value += string(evt.Rune())
			tf.cursorPos += 1
		}
	}
}

func (tf *TextField) Render(s *Screen) {
	tf.RLock()
	defer tf.RUnlock()

	screenW, screenH := s.displaySize()

	x := tf.x
	y := tf.y

	switch x {
	case CenterX:
		x = (screenW - tf.width) / 2
	}

	switch y {
	case CenterY:
		y = (screenH - 2) / 2
	}

	s.DrawText(x+(tf.width-len(tf.label))/2, y-1, tf.sty, tf.label)
	s.DrawBox(x, y, x+tf.width-1, y+2, tf.sty, false)
	s.DrawText(x+(tf.width-len(tf.value))/2, y+1, tf.sty, tf.value)

	if tf.active {
		// Draw selected character with gray background
		ch := " "

		if len(tf.value) > 0 && tf.cursorPos < len(tf.value) {
			ch = string(tf.value[tf.cursorPos])
		}

		selectedSty := tf.sty.Background(tcell.ColorGray)
		cursorX := x + (tf.width-len(tf.value))/2 + tf.cursorPos

		if len(tf.value) == 0 {
			cursorX -= 1
		}

		s.DrawText(cursorX, y+1, selectedSty, ch)
	}
}
