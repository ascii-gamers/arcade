package arcade

import "sync"

type BaseComponent struct {
	sync.RWMutex
	delegate ComponentDelegate
}

func (c *BaseComponent) SetDelegate(d ComponentDelegate) {
	c.Lock()
	c.delegate = d
	c.Unlock()
}
