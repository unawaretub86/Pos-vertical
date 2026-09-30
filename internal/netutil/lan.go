// Package netutil holds small network helpers for LAN access.
package netutil

import "net"

// pickPrivateIPv4 returns the first private IPv4 (10/8, 172.16/12, 192.168/16)
// in ips, or "" if none. Loopback and link-local are never private here.
func pickPrivateIPv4(ips []net.IP) string {
	for _, ip := range ips {
		v4 := ip.To4()
		if v4 == nil {
			continue
		}
		switch {
		case v4[0] == 10,
			v4[0] == 172 && v4[1] >= 16 && v4[1] <= 31,
			v4[0] == 192 && v4[1] == 168:
			return v4.String()
		}
	}
	return ""
}

// LocalIPv4 returns this machine's private LAN IPv4, or "" if not detected.
func LocalIPv4() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok {
			ips = append(ips, ipn.IP)
		}
	}
	return pickPrivateIPv4(ips)
}
