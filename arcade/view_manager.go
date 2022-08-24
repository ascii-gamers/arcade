package arcade

import (
	"arcade/arcade/message"
	"arcade/arcade/net"
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/gdamore/tcell/v2"
)

type ViewManager struct {
	sync.RWMutex

	screen *Screen

	view      View
	showDebug bool
}

func NewViewManager() *ViewManager {
	mgr := &ViewManager{}
	message.AddListener(mgr.ProcessMessage)
	return mgr
}

func (mgr *ViewManager) ProcessMessage(from interface{}, p interface{}) interface{} {
	mgr.RLock()
	v := mgr.view
	mgr.RUnlock()

	return v.ProcessMessage(from.(*net.Client), p)
}

func (mgr *ViewManager) ProcessEvent(ev interface{}) {
	mgr.RLock()
	v := mgr.view
	mgr.RUnlock()

	if arcade.Distributor || v == nil {
		return
	}

	v.ProcessEvent(ev)
}

func (mgr *ViewManager) SetView(v View) {
	mgr.Lock()

	// Unload existing view
	if mgr.view != nil {
		mgr.view.Unload()
	}

	// Reset screen state
	mgr.screen.Reset()

	// Save view
	mgr.view = v
	mgr.view.Init()

	mgr.Unlock()

	// Render
	mgr.RequestRender()
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
				mgr.RLock()
				mgr.view.Unload()
				mgr.RUnlock()

				quit()
			case tcell.KeyCtrlD:
				mgr.ToggleDebugPanel()

				mgr.screen.Reset()
				mgr.RequestRender()
				continue
			case tcell.KeyCtrlQ:
				arcade.Server.Network.SetDropRate(1)
				continue
			case tcell.KeyCtrlW:
				arcade.Server.Network.SetDropRate(0.5)
				continue
			case tcell.KeyCtrlE:
				arcade.Server.Network.SetDropRate(0.1)
				continue
			case tcell.KeyCtrlR:
				arcade.Server.Network.SetDropRate(0)
				continue
			}
		}

		// Send event to current view
		mgr.ProcessEvent(ev)
	}
}

func (mgr *ViewManager) RequestRender() {
	displayWidth, displayHeight := mgr.screen.displaySize()
	width, height := mgr.screen.Size()

	mgr.RLock()
	showDebug := mgr.showDebug
	mgr.RUnlock()

	if showDebug {
		mgr.screen.Reset()
	}

	if width < displayWidth || height < displayHeight {
		warning := "Please make your terminal window larger!"
		mgr.screen.DrawText((displayWidth-len(warning))/2, displayHeight/2-1, tcell.StyleDefault, warning)
	} else {
		mgr.RLock()
		mgr.view.Render(mgr.screen)
		mgr.RUnlock()
	}

	if showDebug {
		x, y := mgr.screen.offset()
		w, h := mgr.screen.displaySize()

		debugSty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorRed)

		mgr.screen.DrawText(-x, -y, debugSty, "Ctrl-D to hide")

		text100 := "Ctrl-Q to drop 100%"
		mgr.screen.DrawText(-x, -y+1, debugSty, text100)

		text50 := "Ctrl-W to drop 50%"
		mgr.screen.DrawText(-x, -y+2, debugSty, text50)

		text10 := "Ctrl-E to drop 10%"
		mgr.screen.DrawText(-x, -y+3, debugSty, text10)

		text0 := "Ctrl-R to drop 0%"
		mgr.screen.DrawText(-x, -y+4, debugSty, text0)

		switch arcade.Server.Network.GetDropRate() {
		case 0:
			mgr.screen.DrawText(-x+len(text0)+1, -y+4, debugSty, "<--")
		case 1 - math.Sqrt(1-0.1):
			mgr.screen.DrawText(-x+len(text10)+1, -y+3, debugSty, "<--")
		case 1 - math.Sqrt(1-0.5):
			mgr.screen.DrawText(-x+len(text50)+1, -y+2, debugSty, "<--")
		case 1:
			mgr.screen.DrawText(-x+len(text100)+1, -y+1, debugSty, "<--")
		}

		connectedClients := arcade.Server.GetHeartbeatClients()

		i := 0
		for clientID, info := range connectedClients {
			s := fmt.Sprintf("%s: %dms", clientID[:4], info.GetMeanRTT().Milliseconds())
			mgr.screen.DrawText(w+x-len(s), -y+i, debugSty, s)
			i++
		}

		if ip, err := net.GetLocalIP(); err == nil {
			mgr.screen.DrawText(-x, h+y-1, debugSty, fmt.Sprintf("Local IP: %s:%d", ip, arcade.Port))
		}
	}

	mgr.screen.Show()
}

func (mgr *ViewManager) RequestDebugRender() {
	mgr.RLock()

	if !mgr.showDebug {
		mgr.RUnlock()
		return
	}

	mgr.RUnlock()
	mgr.RequestRender()
}

func (mgr *ViewManager) GetHeartbeatMetadata() []byte {
	mgr.RLock()
	metadata := mgr.view.GetHeartbeatMetadata()
	mgr.RUnlock()

	if metadata == nil {
		return nil
	}

	data, err := metadata.MarshalBinary()

	if err != nil {
		panic(err)
	}

	return data
}
