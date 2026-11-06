// Package mailer builds and sends the register-close summary email to the
// business owner (US-053). Sending is automatic — no click required — via
// Gmail SMTP using a single account as both sender and recipient, so the
// owner only has to configure one address + one App Password.
package mailer

import (
	"encoding/base64"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

type PaymentLine struct {
	Method string
	Total  float64
}

// RegisterSummaryData carries everything the summary email needs. It is
// framework-free so BuildRegisterSummaryEmail can be unit tested directly.
type RegisterSummaryData struct {
	BusinessName    string
	Cashier         string
	ClosedAt        time.Time
	ByPaymentMethod []PaymentLine
	CashAbonos      float64 // cash abonos received by the register (US-082)
	CashIn          float64
	CashOut         float64
	Expected        float64
	ActualClose     float64
	Difference      float64
}

var methodLabels = map[string]string{
	"cash":     "Efectivo",
	"card":     "Tarjeta",
	"transfer": "Transferencia",
	"credit":   "Crédito",
	"mixed":    "Mixto",
}

// formatCOP mirrors frontend/lib/format.ts formatCOP: no decimals, "." as
// thousands separator, "$" prefix.
func formatCOP(n float64) string {
	neg := n < 0
	v := int64(n)
	if neg {
		v = -v
	}
	digits := fmt.Sprintf("%d", v)
	var out strings.Builder
	for i, c := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out.WriteByte('.')
		}
		out.WriteRune(c)
	}
	if neg {
		return "-$" + out.String()
	}
	return "$" + out.String()
}

// BuildRegisterSummaryEmail builds the subject and plaintext body sent to the
// owner when a register is closed. Same wording as the previous WhatsApp
// message (US-053 business rule), now delivered by email.
func BuildRegisterSummaryEmail(d RegisterSummaryData) (subject, body string) {
	subject = fmt.Sprintf("Resumen de cierre — %s", d.BusinessName)

	var b strings.Builder
	fmt.Fprintf(&b, "Resumen de cierre — %s\n", d.BusinessName)
	fmt.Fprintf(&b, "Cajero: %s · %s\n\n", d.Cashier, d.ClosedAt.Format("02 Jan 2006 15:04"))
	for _, line := range d.ByPaymentMethod {
		label := methodLabels[line.Method]
		if label == "" {
			label = line.Method
		}
		fmt.Fprintf(&b, "%s: %s\n", label, formatCOP(line.Total))
	}
	fmt.Fprintf(&b, "Abonos en efectivo: %s\n", formatCOP(d.CashAbonos))
	fmt.Fprintf(&b, "Entradas: %s\n", formatCOP(d.CashIn))
	fmt.Fprintf(&b, "Salidas: %s\n", formatCOP(d.CashOut))
	fmt.Fprintf(&b, "Esperado: %s\n", formatCOP(d.Expected))
	fmt.Fprintf(&b, "Real: %s\n", formatCOP(d.ActualClose))
	if d.Difference < 0 {
		fmt.Fprintf(&b, "Faltante: %s", formatCOP(-d.Difference))
	} else {
		fmt.Fprintf(&b, "Diferencia: %s", formatCOP(d.Difference))
	}
	return subject, b.String()
}

// SMTPConfig holds the Gmail credentials used to send the summary email.
type SMTPConfig struct {
	Email       string // Gmail address — From, To and SMTP auth user.
	AppPassword string
}

const gmailSMTPAddr = "smtp.gmail.com:587"

func encodeSubject(s string) string {
	return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(s)) + "?="
}

// Send delivers the summary to the owner's own inbox via Gmail SMTP
// (STARTTLS on port 587, handled by net/smtp.SendMail automatically).
func Send(cfg SMTPConfig, subject, body string) error {
	if cfg.Email == "" || cfg.AppPassword == "" {
		return fmt.Errorf("mailer: owner email not configured")
	}
	auth := smtp.PlainAuth("", cfg.Email, cfg.AppPassword, "smtp.gmail.com")
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"utf-8\"\r\n\r\n%s",
		cfg.Email, cfg.Email, encodeSubject(subject), body,
	)
	return smtp.SendMail(gmailSMTPAddr, auth, cfg.Email, []string{cfg.Email}, []byte(msg))
}
