package arcade

import (
	"encoding"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type SplashView struct {
	View

	mu            sync.RWMutex
	displayFooter bool
	stopTickerCh  chan bool
}

var splashHeader1 = []string{
	"░█████╗░░██████╗░█████╗░██╗██╗",
	"██╔══██╗██╔════╝██╔══██╗██║██║",
	"███████║╚█████╗░██║░░╚═╝██║██║",
	"██╔══██║░╚═══██╗██║░░██╗██║██║",
	"██║░░██║██████╔╝╚█████╔╝██║██║",
	"╚═╝░░╚═╝╚═════╝░░╚════╝░╚═╝╚═╝",
}

var splashHeader2 = []string{
	"░█████╗░██████╗░░█████╗░░█████╗░██████╗░███████╗",
	"██╔══██╗██╔══██╗██╔══██╗██╔══██╗██╔══██╗██╔════╝",
	"███████║██████╔╝██║░░╚═╝███████║██║░░██║█████╗░░",
	"██╔══██║██╔══██╗██║░░██╗██╔══██║██║░░██║██╔══╝░░",
	"██╔══██║██╔══██╗██║░░██╗██╔══██║██║░░██║██╔══╝░░",
	"██║░░██║██║░░██║╚█████╔╝██║░░██║██████╔╝███████╗",
	"╚═╝░░╚═╝╚═╝░░╚═╝░╚════╝░╚═╝░░╚═╝╚═════╝░╚══════╝",
}

var splashFooter = "Press any key to start"

func NewSplashView() *SplashView {
	view := &SplashView{
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

				arcade.ViewManager.RequestRender()
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
		arcade.ViewManager.SetView(NewGamesListView())
		// arcade.ViewManager.SetView(NewUsernameView())
	}
}

func (v *SplashView) ProcessMessage(from *Client, p interface{}) interface{} {
	return nil
}

func (v *SplashView) Render(s *Screen) {
	width, _ := s.displaySize()

	// Green text on default background
	sty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen)

	// Draw ASCII ARCADE header
	header1Y := 3
	header2Y := 10

	header1X := (width - utf8.RuneCountInString(splashHeader1[0])) / 2
	header2X := (width - utf8.RuneCountInString(splashHeader2[0])) / 2

	for i := range splashHeader1 {
		s.DrawText(header1X, i+header1Y, sty, splashHeader1[i])
	}

	for i := range splashHeader2 {
		s.DrawText(header2X, i+header2Y, sty, splashHeader2[i])
	}

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
