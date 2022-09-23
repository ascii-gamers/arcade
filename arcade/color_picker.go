package arcade

import (
	"github.com/gdamore/tcell/v2"
)

const COLOR_PICKER_COLS = 4
const COLOR_PICKER_ROWS = 2

type ColorPicker struct {
	BaseComponent

	x, y                     int
	cursorCol, cursorRow     int
	selectedCol, selectedRow int
	active                   bool
}

func NewColorPicker(x, y int) *ColorPicker {
	return &ColorPicker{
		x:           x,
		y:           y,
		selectedCol: -1,
		selectedRow: -1,
	}
}

func (cp *ColorPicker) Focus() {
	cp.Lock()
	defer cp.Unlock()

	cp.active = true
}

func (cp *ColorPicker) ProcessEvent(evt interface{}) {
	cp.Lock()
	defer cp.Unlock()

	switch evt := evt.(type) {
	case *tcell.EventKey:
		switch evt.Key() {
		case tcell.KeyDown:
			if cp.cursorRow == COLOR_PICKER_ROWS-1 {
				if cp.delegate.NavigateForward() {
					cp.active = false
				}

				break
			}

			cp.cursorRow += 1
		case tcell.KeyUp:
			if cp.cursorRow == 0 {
				if cp.delegate.NavigateBackward() {
					cp.active = false
				}

				break
			}

			cp.cursorRow -= 1
		case tcell.KeyLeft:
			if cp.cursorCol == 0 {
				break
			}

			cp.cursorCol -= 1
		case tcell.KeyRight:
			if cp.cursorCol == COLOR_PICKER_COLS-1 {
				break
			}

			cp.cursorCol += 1
		case tcell.KeyEnter:
			cp.selectedCol = cp.cursorCol
			cp.selectedRow = cp.cursorRow
		}
	}
}

func (cp *ColorPicker) Render(s *Screen) {
	cp.RLock()
	defer cp.RUnlock()

	screenW, screenH := s.displaySize()

	startX := cp.x
	startY := cp.y

	componentW := COLOR_PICKER_COLS*4 - 1
	componentH := COLOR_PICKER_ROWS*4 + 1

	switch startX {
	case CenterX:
		startX = (screenW - componentW) / 2
	}

	switch startY {
	case CenterY:
		startY = (screenH - componentH) / 2
	}

	for x := 0; x < COLOR_PICKER_COLS; x++ {
		for y := 0; y < COLOR_PICKER_ROWS; y++ {
			color := TRON_COLORS[y*4+x]
			sty := tcell.StyleDefault.Background(tcell.ColorNames[color])
			s.DrawEmpty(startX+x*4, startY+y*4, startX+x*4+2, startY+y*4+2, sty)

			if cp.active && cp.cursorCol == x && cp.cursorRow == y {
				borderSty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen)
				s.DrawBox(startX+x*4-1, startY+y*4-1, startX+x*4+2+1, startY+y*4+2+1, borderSty, false)
			}
		}
	}

	// Use a second loop to ensure selected border is drawn on top of cursor border
	for x := 0; x < COLOR_PICKER_COLS; x++ {
		for y := 0; y < COLOR_PICKER_ROWS; y++ {
			if cp.selectedCol == x && cp.selectedRow == y {
				borderSty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorTeal)
				s.DrawBox(startX+x*4-1, startY+y*4-1, startX+x*4+2+1, startY+y*4+2+1, borderSty, false)
			}
		}
	}
}
