package arcade

import (
	"sync"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type Screen struct {
	tcell.Screen
	sync.RWMutex
}

type CursorStyle int

const (
	displayWidth  = 80
	displayHeight = 24

	CenterX = 100000
	CenterY = 100001
)

func (s *Screen) displaySize() (int, int) {
	return displayWidth, displayHeight
}

func (s *Screen) Size() (int, int) {
	s.RLock()
	defer s.RUnlock()

	return s.Screen.Size()
}

func (s *Screen) Clear() {
	s.Lock()
	defer s.Unlock()

	s.Screen.Clear()
}

func (s *Screen) ClearContent() {
	sty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen)
	displayWidth, displayHeight := s.displaySize()
	s.DrawEmpty(1, 1, displayWidth-2, displayHeight-2, sty)
}

func (s *Screen) offset() (int, int) {
	currentWidth, currentHeight := s.Size()
	displayWidth, displayHeight := s.displaySize()

	return (currentWidth - displayWidth) / 2, (currentHeight - displayHeight) / 2
}

func (s *Screen) DrawBlockText(x, y int, style tcell.Style, text string, big bool) {
	t := generateText(text, big)
	w, h := s.displaySize()

	switch x {
	case CenterX:
		x = (w - utf8.RuneCountInString(t[0])) / 2
	}

	switch y {
	case CenterY:
		y = (h - len(t)) / 2
	}

	for i := range t {
		s.DrawText(x, i+y, style, t[i])
	}
}

func (s *Screen) DrawText(x, y int, style tcell.Style, text string) {
	startX, startY := s.offset()
	w, h := s.displaySize()

	switch x {
	case CenterX:
		x = (w - utf8.RuneCountInString(text)) / 2
	}

	switch y {
	case CenterY:
		y = (h - 1) / 2
	}

	row := y
	col := x

	for _, r := range text {
		s.SetContent(startX+col, startY+row, r, nil, style)
		col++

		if r == '\n' {
			row++
			col = x
		}
	}
}

func (s *Screen) DrawEmpty(x1, y1, x2, y2 int, style tcell.Style) {
	startX, startY := s.offset()

	for row := y1; row <= y2; row++ {
		for col := x1; col <= x2; col++ {
			s.SetContent(startX+col, startY+row, ' ', nil, style)
		}
	}
}

func (s *Screen) DrawLine(x1, y1, x2, y2 int, style tcell.Style, thick bool) {
	startX, startY := s.offset()

	vertical := '┃'
	horizontal := '━'

	if thick {
		vertical = '║'
		horizontal = '═'
	}

	if x1 == x2 {
		for row := y1; row <= y2; row++ {
			s.SetContent(startX+x1, startY+row, vertical, nil, style)
		}
	} else if y1 == y2 {
		for col := x1; col <= x2; col++ {
			s.SetContent(startX+col, startY+y1, horizontal, nil, style)
		}
	}
}

func (s *Screen) DrawBox(x1, y1, x2, y2 int, style tcell.Style, thicker bool) {
	startX, startY := s.offset()

	vertical := '┃'
	horizontal := '━'
	topLeft := '┏'
	topRight := '┓'
	bottomLeft := '┗'
	bottomRight := '┛'

	if thicker {
		vertical = '║'
		horizontal = '═'
		topLeft = '╔'
		topRight = '╗'
		bottomLeft = '╚'
		bottomRight = '╝'
	}

	for row := y1; row <= y2; row++ {
		s.SetContent(startX+x1, startY+row, vertical, nil, style)
		s.SetContent(startX+x2, startY+row, vertical, nil, style)
	}

	for col := x1 + 1; col < x2; col++ {
		s.SetContent(startX+col, startY+y1, horizontal, nil, style)
		s.SetContent(startX+col, startY+y2, horizontal, nil, style)
	}

	s.SetContent(startX+x1, startY+y1, topLeft, nil, style)
	s.SetContent(startX+x2, startY+y1, topRight, nil, style)
	s.SetContent(startX+x1, startY+y2, bottomLeft, nil, style)
	s.SetContent(startX+x2, startY+y2, bottomRight, nil, style)
}

func (s *Screen) Reset() {
	// Set default text style
	sty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen)
	s.SetStyle(sty)

	// Clear screen
	s.Clear()

	// Set black background
	s.Fill(' ', sty)

	// Draw border around screen
	width, height := s.Size()
	displayWidth, displayHeight := s.displaySize()

	if width >= displayWidth+2 && height >= displayHeight+2 {
		s.DrawBox(0, 0, displayWidth-1, displayHeight-1, sty, true)
	}
}
