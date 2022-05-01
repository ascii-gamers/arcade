package arcade

type View interface {
	Init()
	ProcessEvent(ev interface{})
	ProcessMessage(from *Client, p interface{}) interface{}
	Render(s *Screen)
	Unload()
}
