package arcade

type ComponentDelegate interface {
	NavigateForward() bool
	NavigateBackward() bool
}
