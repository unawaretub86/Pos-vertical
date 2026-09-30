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
