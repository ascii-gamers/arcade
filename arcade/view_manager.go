package arcade

import (
	"os"

	"github.com/gdamore/tcell/v2"
)

type ViewManager struct {
	screen *Screen
	view   View
}

func NewViewManager() *ViewManager {
	return &ViewManager{}
}

func (mgr *ViewManager) SetView(v View) {
	// Set default text style
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	mgr.screen.SetStyle(defStyle)

	// Clear screen
	mgr.screen.Clear()

	// Save view
	mgr.view = v
}

func (mgr *ViewManager) Start(v View) {
	s, err := tcell.NewScreen()
	mgr.screen = &Screen{s}

	if err != nil {
		panic(err)
	}

	if err := mgr.screen.Init(); err != nil {
		panic(err)
	}

	// Set first view
	mgr.SetView(v)

	quit := func() {
		mgr.screen.Fini()
		os.Exit(0)
	}

	for {
		// Update screen
		mgr.view.Render(mgr.screen)
		mgr.screen.Show()

		// Poll event
		ev := mgr.screen.PollEvent()

		// Process event
		switch ev := ev.(type) {
		case *tcell.EventResize:
			mgr.screen.Sync()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				quit()
			}
		}

		mgr.view.ProcessEvent(ev)
	}
}
