package arcade

import (
	"github.com/gdamore/tcell/v2"
)

type Screen struct {
	tcell.Screen
}

type CursorStyle int

func (s *Screen) DrawText(x, y int, style tcell.Style, text string) {
	row := y
	col := x

	for _, r := range []rune(text) {
		s.SetContent(col, row, r, nil, style)
		col++

		if r == '\n' {
			row++
			col = x
		}
	}
}

func (s *Screen) DrawEmpty(x1, y1, x2, y2 int, style tcell.Style) {
	for row := y1; row <= y2; row++ {
		for col := x1; col <= x2; col++ {
			s.SetContent(col, row, ' ', nil, style)
		}
	}
}

func (s *Screen) DrawLine(x1, y1, x2, y2 int, style tcell.Style, thick bool) {
	vertical := '┃'
	horizontal := '━'

	if thick {
		vertical = '║'
		horizontal = '═'
	}

	if x1 == x2 {
		for row := y1; row <= y2; row++ {
			s.SetContent(x1, row, vertical, nil, style)
		}
	} else if y1 == y2 {
		for col := x1; col <= x2; col++ {
			s.SetContent(col, y1, horizontal, nil, style)
		}
	}
}

func (s *Screen) DrawBox(x1, y1, x2, y2 int, style tcell.Style, thicker bool) {
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
		s.SetContent(x1, row, vertical, nil, style)
		s.SetContent(x2, row, vertical, nil, style)
	}

	for col := x1 + 1; col < x2; col++ {
		s.SetContent(col, y1, horizontal, nil, style)
		s.SetContent(col, y2, horizontal, nil, style)
	}

	s.SetContent(x1, y1, topLeft, nil, style)
	s.SetContent(x2, y1, topRight, nil, style)
	s.SetContent(x1, y2, bottomLeft, nil, style)
	s.SetContent(x2, y2, bottomRight, nil, style)
}