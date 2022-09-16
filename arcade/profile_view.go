package arcade

import (
	"arcade/arcade/net"
	"encoding"
	"sync"

	"github.com/gdamore/tcell/v2"
)

type ProfileView struct {
	View
	mgr *ViewManager

	sync.RWMutex

	components     []Component
	componentIndex int
}

func NewProfileView(mgr *ViewManager) *ProfileView {
	view := &ProfileView{
		mgr: mgr,
	}

	view.components = []Component{
		NewTextField(view, CenterX, 7, 30, "Pick a username"),
		NewColorPicker(view, CenterX, 11),
		NewButton(view, CenterX, 19, 20, "CONTINUE", func() {
			mgr.SetView(NewGamesListView(mgr))
		}),
	}

	view.components[0].Focus()
	return view
}

func (v *ProfileView) Init() {
}

func (v *ProfileView) ProcessEvent(evt interface{}) {
	v.components[v.componentIndex].ProcessEvent(evt)
}

func (v *ProfileView) GetHeartbeatMetadata() encoding.BinaryMarshaler {
	return nil
}

func (v *ProfileView) ProcessMessage(from *net.Client, p interface{}) interface{} {
	return nil
}

func (v *ProfileView) Render(s *Screen) {
	s.Clear()

	for _, c := range v.components {
		c.Render(s)
	}

	// Green text on default background
	sty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen)
	s.DrawBlockText(CenterX, 2, sty, "ASCII ARCADE", false)
}

func (v *ProfileView) Unload() {
}

func (v *ProfileView) NavigateForward() bool {
	v.Lock()
	defer v.Unlock()

	if v.componentIndex == len(v.components)-1 {
		return false
	}

	v.componentIndex += 1
	v.components[v.componentIndex].Focus()

	return true
}

func (v *ProfileView) NavigateBackward() bool {
	v.Lock()
	defer v.Unlock()

	if v.componentIndex == 0 {
		return false
	}

	v.componentIndex -= 1
	v.components[v.componentIndex].Focus()

	return true
}
