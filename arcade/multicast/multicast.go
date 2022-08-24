package multicast

import (
	"encoding/json"
	"net"
	"sync"
)

const multicastDiscoveryNetwork = "udp"
const multicastDiscoveryAddress = "224.6.8.24:4445"
const maxDatagramSize = 8192

var multicastConn *net.UDPConn
var multicastConnMu sync.Mutex

func Discover(addr, id string) {
	multicastConnMu.Lock()
	defer multicastConnMu.Unlock()

	if multicastConn == nil {
		addr, err := net.ResolveUDPAddr(multicastDiscoveryNetwork, multicastDiscoveryAddress)

		if err != nil {
			panic(err)
		}

		multicastConn, err = net.DialUDP(multicastDiscoveryNetwork, nil, addr)

		if err != nil {
			panic(err)
		}
	}

	msg := MulticastDiscoveryMessage{
		Addr: addr,
		ID:   id,
	}

	data, _ := json.Marshal(msg)
	multicastConn.Write(data)
}

// Listen binds to a UDP multicast address and responds with a discovery packet.
func Listen(selfID string, delegate MulticastDiscoveryDelegate) {
	// Parse the string address
	addr, err := net.ResolveUDPAddr(multicastDiscoveryNetwork, multicastDiscoveryAddress)

	if err != nil {
		panic(err)
	}

	// Open up a connection
	conn, err := net.ListenMulticastUDP(multicastDiscoveryNetwork, nil, addr)

	if err != nil {
		panic(err)
	}

	conn.SetReadBuffer(maxDatagramSize)

	// Loop forever reading from the socket
	for {
		buffer := make([]byte, maxDatagramSize)
		numBytes, _, err := conn.ReadFromUDP(buffer)

		if err != nil {
			panic(err)
		}

		var msg MulticastDiscoveryMessage
		json.Unmarshal(buffer[:numBytes], &msg)

		if msg.ID == selfID {
			continue
		}

		delegate.ClientDiscovered(msg.Addr, msg.ID)
	}
}
