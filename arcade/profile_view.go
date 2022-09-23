package arcade

import (
	"arcade/arcade/net"
	"encoding"

	"github.com/gdamore/tcell/v2"
)

type ProfileView struct {
	BaseView
	View
}

func NewProfileView(mgr *ViewManager) *ProfileView {
	v := &ProfileView{
		BaseView: NewBaseView(mgr),
	}

	v.SetComponents(v, []Component{
		NewTextField(CenterX, 7, 30, "Pick a username"),
		NewColorPicker(CenterX, 11),
		NewButton(CenterX, 19, 20, "CONTINUE", func() {
			mgr.SetView(NewGamesListView(mgr))
		}),
	})

	return v
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
