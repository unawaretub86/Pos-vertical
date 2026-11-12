package main

import (
	"strings"
	"testing"
	"time"
)

func at(zone string, secs int) time.Time {
	return time.Date(2026, 11, 12, 20, 30, 0, 0, time.FixedZone(zone, secs))
}

// R-162 T-1: UTC-5 logs the offset and no warning.
func TestTimeZoneNotice_T1_Bogota(t *testing.T) {
	info, warn := timeZoneNotice(at("COT", -18000))
	if !strings.Contains(info, "UTC-05:00") {
		t.Errorf("info = %q, want UTC-05:00", info)
	}
	if warn != "" {
		t.Errorf("warn = %q, want empty", warn)
	}
}

// R-162 T-2: UTC warns and names the offset.
func TestTimeZoneNotice_T2_UTC(t *testing.T) {
	info, warn := timeZoneNotice(time.Date(2026, 11, 12, 20, 30, 0, 0, time.UTC))
	if !strings.Contains(info, "UTC+00:00") {
		t.Errorf("info = %q, want UTC+00:00", info)
	}
	if warn == "" || !strings.Contains(warn, "+00:00") {
		t.Errorf("warn = %q, want a warning containing +00:00", warn)
	}
}

// R-162 T-3: minutes are formatted.
func TestTimeZoneNotice_T3_Minutes(t *testing.T) {
	if _, warn := timeZoneNotice(at("x", -14400)); !strings.Contains(warn, "-04:00") {
		t.Errorf("warn = %q, want -04:00", warn)
	}
	if _, warn := timeZoneNotice(at("x", 19800)); !strings.Contains(warn, "+05:30") {
		t.Errorf("warn = %q, want +05:30", warn)
	}
}

// R-162 T-4: only the offset matters, never the zone name.
func TestTimeZoneNotice_T4_OffsetNotName(t *testing.T) {
	if _, warn := timeZoneNotice(at("UTC", -18000)); warn != "" {
		t.Errorf("zone named UTC at -05:00: warn = %q, want empty", warn)
	}
	if _, warn := timeZoneNotice(at("COT", 0)); warn == "" {
		t.Error("zone named COT at +00:00: want a warning")
	}
}

// R-162 T-5: the result follows the injected instant, not the host zone.
func TestTimeZoneNotice_T5_Injected(t *testing.T) {
	_, w1 := timeZoneNotice(at("x", -18000))
	_, w2 := timeZoneNotice(at("x", 0))
	if w1 != "" || w2 == "" {
		t.Errorf("w1 = %q, w2 = %q; want empty then non-empty", w1, w2)
	}
}

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
