package arcade

import (
	"github.com/gdamore/tcell/v2"
)

type TextAlignment int

const (
	AlignLeft TextAlignment = iota
	AlignCenter
	AlignRight
)

type TextFieldOptions struct {
	LayoutOptions

	Alignment    TextAlignment
	Border       bool
	Label        string
	LabelPadding int
}

type TextField struct {
	BaseComponent

	contentX, contentY int
	width, height      int
	sty                tcell.Style
	cursorPos          int
	value              string
	active             bool
	TextFieldOptions
}

func NewTextField(opts TextFieldOptions) *TextField {
	tf := &TextField{
		sty:              tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen),
		TextFieldOptions: opts,
	}

	if tf.Border {
		tf.width = opts.ContentWidth + 4
		tf.height = 3
	} else {
		tf.width = opts.ContentWidth
		tf.height = 1
	}

	if opts.X >= CenterXPlaceholder {
		tf.X = (opts.X - CenterXPlaceholder - tf.width) / 2
	}

	if opts.Y >= CenterYPlaceholder {
		tf.Y = (opts.Y - CenterYPlaceholder - tf.height) / 2
	}

	if tf.Border {
		tf.contentX = tf.X + 2
		tf.contentY = tf.Y + 1
	} else {
		tf.contentX = tf.X
		tf.contentY = tf.Y
	}

	return tf
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
			if len(tf.value) >= tf.width {
				break
			}

			tf.value += string(evt.Rune())
			tf.cursorPos += 1
		}
	}
}

func (tf *TextField) Render(s *Screen) {
	tf.RLock()
	defer tf.RUnlock()

	if tf.Border {
		s.DrawBox(tf.X, tf.Y, tf.X+tf.width-1, tf.Y+tf.height-1, tf.sty, false)
	}

	switch tf.Alignment {
	case AlignLeft:
		s.DrawAlignedText(tf.contentX-tf.LabelPadding, tf.contentY, tf.sty, tf.Label, AlignRight)
		s.DrawAlignedText(tf.contentX, tf.contentY, tf.sty, tf.value, AlignLeft)
	case AlignCenter:
		s.DrawAlignedText(tf.contentX+tf.ContentWidth/2, tf.contentY, tf.sty, tf.value, AlignCenter)
		s.DrawAlignedText(tf.contentX+tf.ContentWidth/2, tf.Y-1, tf.sty, tf.Label, AlignCenter)
	}

	if tf.active {
		// Draw selected character with gray background
		ch := " "

		if len(tf.value) > 0 && tf.cursorPos < len(tf.value) {
			ch = string(tf.value[tf.cursorPos])
		}

		selectedSty := tf.sty.Background(tcell.ColorGray)
		cursorX := tf.contentX + tf.cursorPos

		if tf.Alignment == AlignCenter {
			cursorX = tf.contentX + tf.ContentWidth/2 - len(tf.value)/2 + tf.cursorPos
		}

		s.DrawText(cursorX, tf.contentY, selectedSty, ch)
	}
}
