package arcade

import "sync"

type BaseView struct {
	sync.RWMutex
	mgr *ViewManager

	components     []Component
	componentIndex int
}

func NewBaseView(mgr *ViewManager) BaseView {
	return BaseView{
		mgr:        mgr,
		components: make([]Component, 0),
	}
}

func (v *BaseView) SetComponents(d ComponentDelegate, components []Component) {
	v.Lock()
	defer v.Unlock()

	v.components = components

	for _, c := range v.components {
		c.SetDelegate(d)
	}

	if len(components) > 0 {
		v.components[0].Focus()
	}
}

func (v *BaseView) NavigateForward() bool {
	v.Lock()
	defer v.Unlock()

	if v.componentIndex == len(v.components)-1 {
		return false
	}

	v.componentIndex += 1
	v.components[v.componentIndex].Focus()

	return true
}

func (v *BaseView) NavigateBackward() bool {
	v.Lock()
	defer v.Unlock()

	if v.componentIndex == 0 {
		return false
	}

	v.componentIndex -= 1
	v.components[v.componentIndex].Focus()

	return true
}
