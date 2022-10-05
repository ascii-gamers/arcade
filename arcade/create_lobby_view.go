package arcade

import (
	"arcade/arcade/net"

	"github.com/gdamore/tcell/v2"
)

type CreateLobbyView struct {
	BaseView
	View

	nameField        *TextField
	capacitySelector *HorizontalSelector[int]
}

func NewCreateLobbyView(mgr *ViewManager) *CreateLobbyView {
	v := &CreateLobbyView{
		BaseView: NewBaseView(mgr),
	}

	v.nameField = NewTextField(CenterX, 6, 30, "Lobby Name")
	v.capacitySelector = NewHorizontalSelector(CenterX, 11, 7, "Capacity", []int{2, 3, 4, 5, 6, 7, 8})

	v.SetComponents(v, []Component{
		v.nameField,
		v.capacitySelector,
		NewButton(15, 19, 20, "BACK", func() {

		}),
		NewButton(45, 19, 20, "CONTINUE", func() {

		}),
	})

	return v
}

func (v *CreateLobbyView) Init() {
}

func (v *CreateLobbyView) ProcessEvent(evt interface{}) {
	v.components[v.componentIndex].ProcessEvent(evt)
}

func (v *CreateLobbyView) ProcessMessage(from *net.Client, p interface{}) interface{} {
	return nil
}

func (v *CreateLobbyView) Render(s *Screen) {
	s.Clear()

	for _, c := range v.components {
		c.Render(s)
	}

	// Green text on default background
	sty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen)
	s.DrawBlockText(CenterX, 1, sty, "CREATE GAME", false)
}

func (v *CreateLobbyView) Unload() {
}
