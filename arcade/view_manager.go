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

func (mgr *ViewManager) ProcessEvent(ev interface{}) {
	if distributor {
		return
	}

	mgr.view.ProcessEvent(ev)
}

func (mgr *ViewManager) SetView(v View) {
	// Unload existing view
	if mgr.view != nil {
		mgr.view.Unload()
	}

	// Reset screen state
	mgr.screen.Reset()

	// Save view
	mgr.view = v
	mgr.view.Init()
}

func (mgr *ViewManager) Start(v View) {
	s, err := tcell.NewScreen()

	if err != nil {
		panic(err)
	}

	mgr.screen = &Screen{Screen: s}

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
		mgr.RequestRender()

		// Poll event
		ev := mgr.screen.PollEvent()

		// Process event
		switch ev := ev.(type) {
		case *tcell.EventResize:
			mgr.screen.Reset()
			mgr.RequestRender()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				server.Network.SendNeighbors(NewDisconnectMessage())
				quit()
			}
		}

		// Send event to current view
		mgr.ProcessEvent(ev)
	}
}

func (mgr *ViewManager) RequestRender() {
	displayWidth, displayHeight := mgr.screen.displaySize()
	width, height := mgr.screen.Size()

	if width < displayWidth || height < displayHeight {
		warning := "Please make your terminal window larger!"
		mgr.screen.DrawText((displayWidth-len(warning))/2, displayHeight/2-1, tcell.StyleDefault, warning)
	} else {
		mgr.view.Render(mgr.screen)
	}

	mgr.screen.Show()
}
