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

	v.nameField = NewTextField(TextFieldOptions{
		LayoutOptions: LayoutOptions{
			X:            CenterX(mgr.screen.GetWidth()),
			Y:            6,
			ContentWidth: 24,
		},
		Alignment:    AlignLeft,
		Border:       false,
		Label:        "Lobby Name",
		LabelPadding: 2,
	})

	v.capacitySelector = NewHorizontalSelector(HorizontalSelectorOptions[int]{
		LayoutOptions: LayoutOptions{
			X:            CenterX(mgr.screen.GetWidth()),
			Y:            11,
			ContentWidth: 7,
		},
		Label:  "Capacity",
		Values: []int{2, 3, 4, 5, 6, 7, 8},
	})

	v.SetComponents(v, []Component{
		v.nameField,
		v.capacitySelector,
		NewButton(15, 20, 20, "BACK", func() {

		}),
		NewButton(45, 20, 20, "CONTINUE", func() {

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
	s.DrawBlockText(CenterX(s.GetWidth()), 1, sty, "CREATE GAME", false)
}

func (v *CreateLobbyView) Unload() {
}
