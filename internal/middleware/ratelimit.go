package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	loginMaxFailures = 10
	loginWindow      = 10 * time.Minute
	loginLockout     = 5 * time.Minute
)

type loginAttempts struct {
	fails       int
	first       time.Time
	lockedUntil time.Time
}

// LoginLimiter locks an IP for a few minutes after repeated failed logins.
// It is keyed by IP, not by username, so nobody on the network can lock the
// admin out by guessing their name. State is in memory: a restart clears it.
type LoginLimiter struct {
	mu       sync.Mutex
	attempts map[string]*loginAttempts
	now      func() time.Time
}

func NewLoginLimiter() *LoginLimiter {
	return &LoginLimiter{attempts: map[string]*loginAttempts{}, now: time.Now}
}

// Allowed reports whether ip may try to log in; when locked it also returns
// how long the lock still lasts.
func (l *LoginLimiter) Allowed(ip string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	a := l.attempts[ip]
	if a == nil {
		return true, 0
	}
	if remaining := a.lockedUntil.Sub(l.now()); remaining > 0 {
		return false, remaining
	}
	return true, 0
}

// Fail records a failed login; the loginMaxFailures-th failure inside the
// window starts the lockout.
func (l *LoginLimiter) Fail(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.purge(now)
	a := l.attempts[ip]
	if a == nil || now.Sub(a.first) > loginWindow {
		a = &loginAttempts{first: now}
		l.attempts[ip] = a
	}
	a.fails++
	if a.fails >= loginMaxFailures {
		a.lockedUntil = now.Add(loginLockout)
		a.fails = 0
		a.first = now
	}
}

// Reset clears the counter after a successful login.
func (l *LoginLimiter) Reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, ip)
}

// purge drops entries that can no longer lock anyone; caller holds the mutex.
func (l *LoginLimiter) purge(now time.Time) {
	for ip, a := range l.attempts {
		if now.After(a.lockedUntil) && now.Sub(a.first) > loginWindow {
			delete(l.attempts, ip)
		}
	}
}

// ClientIP returns the peer address. X-Forwarded-For is ignored on purpose:
// the POS has no reverse proxy, so the header could only be forged.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
