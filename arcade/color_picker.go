package arcade

import (
	"github.com/gdamore/tcell/v2"
)

type ColorPickerConfig struct {
	Columns       int
	Rows          int
	TileWidth     int
	TileHeight    int
	VerticalGap   int
	HorizontalGap int
}

type ColorPicker struct {
	BaseComponent
	x, y                     int
	cursorCol, cursorRow     int
	selectedCol, selectedRow int
	active                   bool

	config ColorPickerConfig
}

func NewColorPicker(x, y int) *ColorPicker {
	return &ColorPicker{
		x:           x,
		y:           y,
		selectedCol: -1,
		selectedRow: -1,
		config: ColorPickerConfig{
			Columns:       4,
			Rows:          2,
			TileWidth:     6,
			TileHeight:    3,
			VerticalGap:   1,
			HorizontalGap: 2,
		},
	}
}

func (cp *ColorPicker) SelectedColor() string {
	cp.RLock()
	defer cp.RUnlock()

	return TRON_COLORS[cp.selectedRow*cp.config.Columns+cp.selectedCol]
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
			if cp.cursorRow == cp.config.Rows-1 {
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
			if cp.cursorCol == cp.config.Columns-1 {
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

	startX := cp.x
	startY := cp.y

	componentW := cp.config.Columns*(cp.config.TileWidth+cp.config.HorizontalGap) - 1
	componentH := cp.config.Rows*(cp.config.TileHeight+cp.config.VerticalGap) + 1

	if startX >= CenterXPlaceholder {
		startX = (startX - CenterXPlaceholder - componentW) / 2
	}

	if startY >= CenterYPlaceholder {
		startY = (startY - CenterYPlaceholder - componentH) / 2
	}

	for x := 0; x < cp.config.Columns; x++ {
		for y := 0; y < cp.config.Rows; y++ {
			color := TRON_COLORS[y*cp.config.Columns+x]
			sty := tcell.StyleDefault.Background(tcell.ColorNames[color])
			s.DrawEmpty(startX+x*(cp.config.TileWidth+cp.config.HorizontalGap), startY+y*(cp.config.TileHeight+cp.config.VerticalGap), startX+x*(cp.config.TileWidth+cp.config.HorizontalGap)+(cp.config.TileWidth-1), startY+y*(cp.config.TileHeight+cp.config.VerticalGap)+(cp.config.TileHeight-1), sty)

			if cp.active && cp.cursorCol == x && cp.cursorRow == y {
				borderSty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen)
				s.DrawBox(startX+x*(cp.config.TileWidth+cp.config.HorizontalGap)-1, startY+y*(cp.config.TileHeight+cp.config.VerticalGap)-1, startX+x*(cp.config.TileWidth+cp.config.HorizontalGap)+(cp.config.TileWidth-1)+1, startY+y*(cp.config.TileHeight+cp.config.VerticalGap)+(cp.config.TileHeight-1)+1, borderSty, false)
			}
		}
	}

	// Use a second loop to ensure selected border is drawn on top of cursor border
	for x := 0; x < cp.config.Columns; x++ {
		for y := 0; y < cp.config.Rows; y++ {
			if cp.selectedCol == x && cp.selectedRow == y {
				borderSty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorTeal)
				s.DrawBox(startX+x*(cp.config.TileWidth+cp.config.HorizontalGap)-1, startY+y*(cp.config.TileHeight+cp.config.VerticalGap)-1, startX+x*(cp.config.TileWidth+cp.config.HorizontalGap)+(cp.config.TileWidth-1)+1, startY+y*(cp.config.TileHeight+cp.config.VerticalGap)+(cp.config.TileHeight-1)+1, borderSty, false)
			}
		}
	}
}
