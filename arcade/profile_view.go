package arcade

import (
	"arcade/arcade/net"
	"encoding"

	"github.com/gdamore/tcell/v2"
)

type ProfileView struct {
	BaseView
	View

	nameField   *TextField
	colorPicker *ColorPicker
}

func NewProfileView(mgr *ViewManager) *ProfileView {
	v := &ProfileView{
		BaseView: NewBaseView(mgr),
	}

	v.nameField = NewTextField(CenterX, 7, 30, "Pick a username")
	v.colorPicker = NewColorPicker(CenterX, 11)

	v.SetComponents(v, []Component{
		v.nameField,
		v.colorPicker,
		NewButton(CenterX, 19, 20, "CONTINUE", func() {
			profile := &Profile{
				Name:  v.nameField.value,
				Color: v.colorPicker.SelectedColor(),
			}
			profile.Save()

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
