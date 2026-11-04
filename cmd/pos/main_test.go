package main

import "testing"

// Contrato US-054 T-1 (AC-1, AC-2).
func TestListenAddr_T1(t *testing.T) {
	if got := listenAddr(false); got != "127.0.0.1:8080" {
		t.Errorf("listenAddr(false) = %q, want 127.0.0.1:8080", got)
	}
	if got := listenAddr(true); got != "0.0.0.0:8080" {
		t.Errorf("listenAddr(true) = %q, want 0.0.0.0:8080", got)
	}
}

// Contrato US-054 T-2 (AC-3): el navegador siempre va a localhost.
func TestBrowserURL_T2(t *testing.T) {
	if got := browserURL(); got != "http://localhost:8080" {
		t.Errorf("browserURL() = %q, want http://localhost:8080", got)
	}
}

// R-71: --port / POS_PORT override; invalid values fall back to 8080.
func TestResolvePort_T3(t *testing.T) {
	cases := []struct{ flag, env, want string }{
		{"", "", "8080"},
		{"", "9090", "9090"},
		{"7070", "9090", "7070"}, // the flag wins over the environment
		{"", "abc", "8080"},
		{"", "0", "8080"},
		{"", "70000", "8080"},
		{"", " 9090 ", "9090"},
	}
	for _, c := range cases {
		if got := resolvePort(c.flag, c.env); got != c.want {
			t.Errorf("resolvePort(%q, %q) = %q, want %q", c.flag, c.env, got, c.want)
		}
	}
}

func TestListenAddrAndBrowserURL_T4_FollowPort(t *testing.T) {
	old := port
	t.Cleanup(func() { port = old })
	port = "9090"
	if got := listenAddr(false); got != "127.0.0.1:9090" {
		t.Errorf("listenAddr(false) = %q, want 127.0.0.1:9090", got)
	}
	if got := listenAddr(true); got != "0.0.0.0:9090" {
		t.Errorf("listenAddr(true) = %q, want 0.0.0.0:9090", got)
	}
	if got := browserURL(); got != "http://localhost:9090" {
		t.Errorf("browserURL() = %q, want http://localhost:9090", got)
	}
}
