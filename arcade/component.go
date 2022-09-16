package arcade

type Component interface {
	Focus()
	ProcessEvent(evt interface{})
	Render(s *Screen)
}
