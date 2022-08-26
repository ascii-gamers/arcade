package multicast

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
)

const multicastDiscoveryNetwork = "udp"
const multicastDiscoveryAddress = "224.6.8.24:4445"
const maxDatagramSize = 8192

var multicastConn *net.UDPConn
var multicastConnMu sync.Mutex

func Discover(addr, id string, port int) {
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

	ip, err := GetLocalIP()

	if err != nil {
		panic(err)
	}

	msg := MulticastDiscoveryMessage{
		Addr: fmt.Sprintf("%s:%d", ip, port),
		ID:   id,
	}

	data, _ := json.Marshal(msg)

	log.Println("Writing to multicast...")
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

	// Start reading from connection
	conn.SetReadBuffer(maxDatagramSize)
	buf := make([]byte, maxDatagramSize)

	// Loop forever reading from the socket
	for {
		n, _, err := conn.ReadFromUDP(buf)

		if err != nil {
			panic(err)
		}

		var msg MulticastDiscoveryMessage
		json.Unmarshal(buf[:n], &msg)

		if msg.ID == selfID {
			log.Println("Multicast discovery of self")
			continue
		}

		log.Println("Multicast discovery", msg.Addr, msg.ID)
		delegate.ClientDiscovered(msg.Addr, msg.ID)
	}
}
