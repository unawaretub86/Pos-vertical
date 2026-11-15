package handler

// US-108 contract §4, T-1 .. T-3: a register close sends exactly one owner
// summary email (from the handler), logs a failed send once, and sends nothing
// when the owner email or App Password is missing.
//
// Fixture: file-backed DB (newShiftEnv), not ":memory:" (R-115): the email
// goroutine takes a second pooled connection.

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/andreyo/pos/internal/mailer"
)

type sentMail struct {
	to  []string
	msg string
}

// stubTransport replaces mailer.Transport. Each call is pushed on the returned
// channel; the stub blocks until release is closed, then returns err.
func stubTransport(t *testing.T, err error) (sent chan sentMail, release func()) {
	t.Helper()
	sent = make(chan sentMail, 4)
	gate := make(chan struct{})
	var once sync.Once
	release = func() { once.Do(func() { close(gate) }) }
	orig := mailer.Transport
	mailer.Transport = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		sent <- sentMail{to: to, msg: string(msg)}
		<-gate
		return err
	}
	t.Cleanup(func() {
		release()
		mailer.Transport = orig
	})
	return sent, release
}

// settleCount waits 300 ms after the first delivery (if any) and returns how
// many were received in total.
func settleCount(sent chan sentMail, first *sentMail) int {
	n := 0
	select {
	case m := <-sent:
		n = 1
		if first != nil {
			*first = m
		}
	case <-time.After(2 * time.Second):
		return 0
	}
	time.Sleep(300 * time.Millisecond)
	return n + len(sent)
}

// closeAs95k opens the register as the cashier with $100.000 and closes it as
// admin counting $95.000.
func closeAs95k(t *testing.T, e *shiftEnv) []byte {
	t.Helper()
	e.openRegister(t, 100000)
	return e.expectStatus(t, http.MethodPost, "/registers/1/close", e.admin, `{"actual_close":95000}`, http.StatusOK)
}

func TestRegisterEmail_T1_ExactlyOneEmail(t *testing.T) {
	e := newShiftEnv(t)
	putSetup(t, e.h, `{"name":"Tienda Uno","owner_email":"dueno@gmail.com","owner_email_app_password":"abcd"}`)
	sent, release := stubTransport(t, nil)

	// The stub blocks until released: the HTTP call returning proves AC-6.
	raw := closeAs95k(t, e)
	var env struct {
		Data shiftSummaryJSON `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("decode close response: %v; body %s", err, raw)
	}
	if d := env.Data.Register.Difference; d == nil || *d != -5000 {
		t.Fatalf("difference = %v, want -5000; body %s", d, raw)
	}

	var first sentMail
	count := settleCount(sent, &first)
	release()
	if count != 1 {
		t.Fatalf("emails sent = %d, want exactly 1", count)
	}
	if len(first.to) != 1 || first.to[0] != "dueno@gmail.com" {
		t.Errorf("recipients = %v, want [dueno@gmail.com]", first.to)
	}
	for _, want := range []string{
		"Cajero: u", "Esperado: $100.000", "Real: $95.000",
		"Faltante: $5.000", "Abonos en efectivo: $0",
	} {
		if !strings.Contains(first.msg, want) {
			t.Errorf("email lacks %q; msg:\n%s", want, first.msg)
		}
	}
}

// syncBuf is a mutex-guarded log sink: the email goroutine writes while the
// test polls.
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func TestRegisterEmail_T2_SendFailureLoggedOnce(t *testing.T) {
	e := newShiftEnv(t)
	putSetup(t, e.h, `{"name":"Tienda Uno","owner_email":"dueno@gmail.com","owner_email_app_password":"abcd"}`)
	sent, release := stubTransport(t, errors.New("smtp down"))
	release() // fail immediately

	buf := &syncBuf{}
	prev := log.Writer()
	log.SetOutput(buf)
	t.Cleanup(func() { log.SetOutput(prev) })

	closeAs95k(t, e)

	const line = "register summary email: smtp down"
	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(buf.String(), line) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond)
	if got := strings.Count(buf.String(), line); got != 1 {
		t.Errorf("log line count = %d, want 1; log:\n%s", got, buf.String())
	}
	if got := len(sent); got != 1 {
		t.Errorf("stub calls = %d, want 1", got)
	}

	var status string
	var actual float64
	if err := e.db.QueryRow(`SELECT status, actual_close FROM registers WHERE id = 1`).Scan(&status, &actual); err != nil {
		t.Fatalf("query register: %v", err)
	}
	if status != "closed" || actual != 95000 {
		t.Errorf("register = %q / %.0f, want closed / 95000", status, actual)
	}
}

func TestRegisterEmail_T3_NotConfiguredSendsNothing(t *testing.T) {
	cases := []struct{ name, setup string }{
		{"no owner email", `{"name":"Tienda Uno"}`},
		{"no app password", `{"name":"Tienda Uno","owner_email":"dueno@gmail.com"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newShiftEnv(t)
			putSetup(t, e.h, c.setup)
			sent, release := stubTransport(t, nil)
			release()

			closeAs95k(t, e)

			time.Sleep(300 * time.Millisecond)
			if got := len(sent); got != 0 {
				t.Errorf("emails sent = %d, want 0", got)
			}
		})
	}
}
