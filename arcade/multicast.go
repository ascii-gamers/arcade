package arcade

import (
	"net"
	"sync"
)

const multicastDiscoveryNetwork = "udp"
const multicastDiscoveryAddress = "224.6.8.24:4445"
const maxDatagramSize = 8192

var multicastConn *net.UDPConn
var multicastConnMu sync.Mutex

func discoverMulticast() {
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

	multicastConn.Write([]byte(arcade.Server.Network.Addr()))
}

// listenMulticast binds to a UDP multicast address and responds with a
// discovery packet.
func listenMulticast() {
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

		addr := string(buffer[:numBytes])

		if addr == arcade.Server.Network.Addr() {
			continue
		}

		arcade.Server.Network.Connect(addr, nil)
	}
}
