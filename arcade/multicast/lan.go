package multicast

import (
	"errors"
	"net"
)

func GetLocalIP() (string, error) {
	ifaces, err := net.Interfaces()

	if err != nil {
		return "", err
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()

		if err != nil {
			return "", err
		}

		for _, a := range addrs {
			switch v := a.(type) {
			case *net.IPNet:
				if i.Name != "en0" || v.IP.To4() == nil {
					continue
				}

				return v.IP.String(), nil
			}

		}
	}

	return "", errors.New("No network interfaces found")
}
