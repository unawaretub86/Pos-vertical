package netutil

import (
	"net"
	"testing"
)

// Contrato US-054 T-4 (AC-6, AC-7, AC-8). Firma esperada: pickPrivateIPv4([]net.IP) string.
func TestPickPrivateIPv4_T4(t *testing.T) {
	ips := func(ss ...string) []net.IP {
		out := make([]net.IP, 0, len(ss))
		for _, s := range ss {
			out = append(out, net.ParseIP(s))
		}
		return out
	}
	cases := []struct {
		name string
		in   []net.IP
		want string
	}{
		{"skips loopback and link-local", ips("127.0.0.1", "169.254.3.4", "192.168.1.15"), "192.168.1.15"},
		{"first private wins, stable order", ips("10.0.0.7", "192.168.1.15"), "10.0.0.7"},
		{"172.15 and 172.32 are public", ips("172.15.0.1", "172.32.0.1", "8.8.8.8"), ""},
		{"172.16 lower bound", ips("172.16.0.1"), "172.16.0.1"},
		{"172.31 upper bound", ips("172.31.255.254"), "172.31.255.254"},
		{"empty", ips(), ""},
		{"nil", nil, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := pickPrivateIPv4(c.in); got != c.want {
				t.Errorf("pickPrivateIPv4 = %q, want %q", got, c.want)
			}
		})
	}
}
