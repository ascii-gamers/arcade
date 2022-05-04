package arcade

import (
	"os"
	"sync"

	"github.com/gdamore/tcell/v2"
)

type ViewManager struct {
	sync.RWMutex

	screen *Screen

	view View

	showDebug bool
}

func NewViewManager() *ViewManager {
	return &ViewManager{}
}

func (mgr *ViewManager) ProcessMessage(from *Client, p interface{}) interface{} {
	return mgr.view.ProcessMessage(from, p)
}

func (mgr *ViewManager) ProcessEvent(ev interface{}) {
	if arcade.Distributor {
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

func (mgr *ViewManager) ToggleDebugPanel() {
	mgr.Lock()
	defer mgr.Unlock()

	mgr.showDebug = !mgr.showDebug
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
			switch ev.Key() {
			case tcell.KeyEscape, tcell.KeyCtrlC:
				arcade.Server.Network.SendNeighbors(NewDisconnectMessage())
				quit()
			case tcell.KeyCtrlD:
				mgr.ToggleDebugPanel()

				mgr.screen.Reset()
				mgr.RequestRender()
			case tcell.KeyCtrlQ:
				arcade.Server.Network.SetDropRate(1)
			case tcell.KeyCtrlW:
				arcade.Server.Network.SetDropRate(0.5)
			case tcell.KeyCtrlE:
				arcade.Server.Network.SetDropRate(0.1)
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

	mgr.RLock()
	showDebug := mgr.showDebug
	mgr.RUnlock()

	if showDebug {
		x, y := mgr.screen.offset()
		debugSty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorRed)

		mgr.screen.DrawText(-x, -y, debugSty, "Ctrl-D to hide")
		mgr.screen.DrawText(-x, -y+1, debugSty, "Ctrl-Q to drop 100%")
		mgr.screen.DrawText(-x, -y+2, debugSty, "Ctrl-W to drop 50%")
		mgr.screen.DrawText(-x, -y+3, debugSty, "Ctrl-E to drop 10%")
	}

	mgr.screen.Show()
}
