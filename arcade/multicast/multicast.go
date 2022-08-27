package multicast

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"golang.org/x/net/ipv4"
)

var multicastConn *net.UDPConn
var multicastGroup = &net.UDPAddr{IP: net.IPv4(224, 0, 0, 250)}
var multicastAddr = &net.UDPAddr{IP: net.IPv4(224, 0, 0, 250), Port: 36824}

func Listen(selfID string, delegate MulticastDiscoveryDelegate, startCh chan error) {
	var err error
	multicastConn, err = net.ListenUDP("udp4", multicastAddr)

	if err != nil {
		startCh <- err
		return
	}

	pc := ipv4.NewPacketConn(multicastConn)

	// TODO: Add en1
	iface, err := net.InterfaceByName("en0")

	if err != nil {
		startCh <- err
		return
	}

	if err := pc.JoinGroup(iface, multicastGroup); err != nil {
		startCh <- err
		return
	}

	if loop, err := pc.MulticastLoopback(); err == nil {
		if !loop {
			if err := pc.SetMulticastLoopback(true); err != nil {
				startCh <- err
				return
			}
		}
	}

	buf := make([]byte, 1024)
	startCh <- nil

	for {
		n, _, err := multicastConn.ReadFrom(buf)

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

		// Who knows why this fixes the problem
		time.Sleep(100 * time.Millisecond)
	}
}

func Discover(addr, id string, port int) {
	if multicastConn == nil {
		return
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

	if _, err := multicastConn.WriteTo(data, multicastAddr); err != nil {
		panic(err)
	}
}
