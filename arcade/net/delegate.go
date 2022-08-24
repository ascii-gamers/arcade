package net

type ClientDelegate interface {
	ClientDisconnected(id string)
}

type NetworkDelegate interface {
	ClientConnected(id string)
	ClientDisconnected(id string)
}
