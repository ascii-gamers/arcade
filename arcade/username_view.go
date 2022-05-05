package arcade

import (
	"encoding"

	"github.com/gdamore/tcell/v2"
)

type UsernameView struct {
	View
}

var username_footer = "Press enter to save"
var uv_input = ""

func NewUsernameView() *UsernameView {
	return &UsernameView{}
}

func (v *UsernameView) Init() {

}

func (v *UsernameView) ProcessEvent(evt interface{}) {
	switch evt := evt.(type) {
	case *tcell.EventKey:
		switch evt.Key() {
		case tcell.KeyRune:
			uv_input += string(evt.Rune())

		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if len(uv_input) > 0 {
				uv_input = uv_input[:len(uv_input)-1]
			}
		case tcell.KeyEnter:
			if uv_input != "" {
				// client.Username = uv_input
				arcade.ViewManager.SetView(NewGamesListView())
			}
		}
	}
}

func (v *UsernameView) ProcessMessage(from *Client, p interface{}) interface{} {
	return nil
}

func (v *UsernameView) Render(s *Screen) {
	width, _ := s.displaySize()

	// Green text on default background
	sty := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen)
	sty_bold := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorLightGreen)

	// Draw box surrounding games list
	s.DrawBox(joinbox_X1, joinbox_Y1, joinbox_X2, joinbox_Y2, sty, true)

	usernameHeader := "Enter username: "
	s.DrawText((width-len(usernameHeader)-5)/2, joinbox_Y1+2, sty, usernameHeader)
	s.DrawText((width-len(usernameHeader)-5)/2+len(usernameHeader), joinbox_Y1+2, sty_bold, uv_input)
	s.DrawEmpty((width-len(usernameHeader)-5)/2+len(usernameHeader)+len(uv_input), joinbox_Y1+2, width/2+(joinbox_X2-joinbox_X1)/2-1, joinbox_Y1+3, sty_bold)

}

func (v *UsernameView) Unload() {
}

func (v *UsernameView) GetHeartbeatMetadata() encoding.BinaryMarshaler {
	return nil
}
