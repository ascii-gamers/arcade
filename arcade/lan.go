package arcade

import (
	"errors"
	"net"
)

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func GetLANIPs() ([]string, error) {
	ifaces, err := net.Interfaces()

	if err != nil {
		return nil, err
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()

		if err != nil {
			return nil, err
		}

		for _, a := range addrs {
			switch v := a.(type) {
			case *net.IPNet:
				if i.Name != "en0" || v.IP.To4() == nil {
					continue
				}

				ip, ipnet, err := net.ParseCIDR(a.String())

				if err != nil {
					continue
				}

				ips := make([]string, 0)

				for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incIP(ip) {
					ips = append(ips, ip.String())
				}

				return ips, nil
			}

		}
	}

	return nil, errors.New("No network interfaces found")
}
