package multicast

type MulticastDiscoveryDelegate interface {
	ClientDiscovered(addr, id string)
}
