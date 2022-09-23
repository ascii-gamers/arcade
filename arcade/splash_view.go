package arcade

import (
	"arcade/arcade/net"
	"encoding"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
)

type SplashView struct {
	View
	mgr *ViewManager

	mu            sync.RWMutex
	displayFooter bool
	stopTickerCh  chan bool
}

var splashFooter = "Press any key to start"

func NewSplashView(mgr *ViewManager) *SplashView {
	view := &SplashView{
		mgr:           mgr,
		displayFooter: true,
		stopTickerCh:  make(chan bool),
	}

	ticker := time.NewTicker(750 * time.Millisecond)

	go func() {
		for {
			select {
			case <-ticker.C:
				view.mu.Lock()
				view.displayFooter = !view.displayFooter
				view.mu.Unlock()

				view.mgr.RequestRender()
			case <-view.stopTickerCh:
				ticker.Stop()
				return
			}
		}
	}()

	return view
}

func (v *SplashView) Init() {

}

func (v *SplashView) ProcessEvent(evt interface{}) {
	switch evt.(type) {
	case *tcell.EventKey:
		if _, err := LoadProfile(); err != nil {
			v.mgr.SetView(NewProfileView(v.mgr))
		} else {
			v.mgr.SetView(NewGamesListView(v.mgr))
		}
	}
}

func (v *SplashView) ProcessMessage(from *net.Client, p interface{}) interface{} {
	return nil
}

func (v *SplashView) Render(s *Screen) {
	width, _ := s.displaySize()

	// Green text on default background
	sty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen)

	// Draw ASCII ARCADE header
	s.DrawBlockText(CenterX, 3, sty, "ASCII", true)
	s.DrawBlockText(CenterX, 10, sty, "ARCADE", true)

	// Draw footer
	v.mu.RLock()
	defer v.mu.RUnlock()

	footerX := (width - len(splashFooter)) / 2
	footerY := 20

	if v.displayFooter {
		s.DrawText(footerX, footerY, sty, splashFooter)
	} else {
		s.DrawEmpty(footerX, footerY, footerX+len(splashFooter), footerY, sty)
	}
}

func (v *SplashView) Unload() {
	v.stopTickerCh <- true
}

func (v *SplashView) GetHeartbeatMetadata() encoding.BinaryMarshaler {
	return nil
}
