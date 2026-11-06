package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andreyo/pos/internal/model"
)

// ErrUnsupported is returned by a transport that cannot run on this OS
// (the Windows print queue on non-Windows servers).
var ErrUnsupported = errors.ErrUnsupported

// Print methods stored in business_config.print_method.
const (
	PrintMethodBrowser = "browser"
	PrintMethodNetwork = "escpos_network"
	PrintMethodWindows = "escpos_windows"
)

const (
	networkDialTimeout  = 3 * time.Second
	networkWriteTimeout = 10 * time.Second
)

// ESC/POS command sequences (Epson command set).
var (
	escInit        = []byte{0x1B, 0x40}       // ESC @
	escCodepage858 = []byte{0x1B, 0x74, 0x13} // ESC t 19 (CP858)
	escAlignLeft   = []byte{0x1B, 0x61, 0x00}
	escAlignCenter = []byte{0x1B, 0x61, 0x01}
	escBoldOn      = []byte{0x1B, 0x45, 0x01}
	escBoldOff     = []byte{0x1B, 0x45, 0x00}
	gsDoubleHeight = []byte{0x1D, 0x21, 0x01} // GS ! 1: double height, normal width
	gsNormalSize   = []byte{0x1D, 0x21, 0x00}
	escFeed3       = []byte{0x1B, 0x64, 0x03} // ESC d 3
	gsCutPartial   = []byte{0x1D, 0x56, 0x01} // GS V 1
)

// cp858 maps the non-ASCII runes a Spanish ticket needs to their CP858 byte.
var cp858 = map[rune]byte{
	'á': 0xA0, 'é': 0x82, 'í': 0xA1, 'ó': 0xA2, 'ú': 0xA3, 'ü': 0x81, 'ñ': 0xA4,
	'Á': 0xB5, 'É': 0x90, 'Í': 0xD6, 'Ó': 0xE0, 'Ú': 0xE9, 'Ü': 0x9A, 'Ñ': 0xA5,
	'¿': 0xA8, '¡': 0xAD, '€': 0xD5,
}

// encodeCP858 converts text to CP858. Unmappable runes become '?'.
func encodeCP858(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		switch {
		case r == '\n':
			out = append(out, 0x0A)
		case r == '\r':
			// dropped: LF alone ends a line
		case r == '\t':
			out = append(out, ' ')
		case r >= 0x20 && r <= 0x7E:
			out = append(out, byte(r))
		default:
			if b, ok := cp858[r]; ok {
				out = append(out, b)
			} else {
				out = append(out, '?')
			}
		}
	}
	return out
}

// escposBuilder accumulates one ticket.
type escposBuilder struct {
	buf   []byte
	width int // columns
}

func (b *escposBuilder) raw(cmd []byte) { b.buf = append(b.buf, cmd...) }

func (b *escposBuilder) line(s string) {
	b.buf = append(b.buf, encodeCP858(s)...)
	b.buf = append(b.buf, 0x0A)
}

func (b *escposBuilder) centered(s string) {
	b.raw(escAlignCenter)
	for _, l := range wrapText(s, b.width) {
		b.line(l)
	}
	b.raw(escAlignLeft)
}

func (b *escposBuilder) boldCentered(s string) {
	b.raw(escBoldOn)
	b.centered(s)
	b.raw(escBoldOff)
}

func (b *escposBuilder) divider() { b.line(strings.Repeat("-", b.width)) }

// cols prints left text and right text on one line, the right one flush to the
// right edge. The left text is truncated so the line never exceeds the width.
func (b *escposBuilder) cols(left, right string) {
	b.line(twoCols(left, right, b.width))
}

func twoCols(left, right string, width int) string {
	rr := []rune(right)
	if len(rr) >= width {
		return string(rr[:width])
	}
	lr := []rune(left)
	room := width - len(rr)
	if len(rr) > 0 {
		room-- // keep a one-space gap before the right column
	}
	if len(lr) > room {
		lr = lr[:room]
	}
	pad := width - len(lr) - len(rr)
	if len(rr) == 0 {
		return string(lr)
	}
	return string(lr) + strings.Repeat(" ", pad) + right
}

// wrapText splits s on newlines and word-wraps each line to width runes.
func wrapText(s string, width int) []string {
	var out []string
	for _, para := range strings.Split(strings.ReplaceAll(s, "\r", ""), "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			continue
		}
		cur := ""
		for _, w := range words {
			for len([]rune(w)) > width {
				if cur != "" {
					out = append(out, cur)
					cur = ""
				}
				r := []rune(w)
				out = append(out, string(r[:width]))
				w = string(r[width:])
			}
			switch {
			case cur == "":
				cur = w
			case len([]rune(cur))+1+len([]rune(w)) <= width:
				cur += " " + w
			default:
				out = append(out, cur)
				cur = w
			}
		}
		if cur != "" {
			out = append(out, cur)
		}
	}
	return out
}

// formatMoney renders pesos as $1.234.567 (no decimals, dot thousands).
func formatMoney(v float64) string {
	n := int64(v + 0.5)
	if v < 0 {
		n = int64(v - 0.5)
	}
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	out := "$" + strings.Join(parts, ".")
	if neg {
		out = "-" + out
	}
	return out
}

func columnsFor(widthMM int) int {
	if widthMM == 80 {
		return 48
	}
	return 32
}

var ticketUnitLabels = map[string]string{"unit": "und", "kg": "kg", "lb": "lb", "liter": "lt"}

var ticketMethodLabels = map[string]string{
	"cash": "Efectivo", "card": "Tarjeta", "transfer": "Transferencia", "credit": "Crédito",
}

func methodLabel(m string) string {
	if l, ok := ticketMethodLabels[m]; ok {
		return l
	}
	return m
}

func ticketQty(q float64, unit string) string {
	label, ok := ticketUnitLabels[unit]
	if !ok {
		label = unit
	}
	n := strconv.FormatFloat(q, 'f', 2, 64)
	if q == float64(int64(q)) {
		n = strconv.FormatInt(int64(q), 10)
	}
	return n + " " + label
}

func (b *escposBuilder) finish() []byte {
	b.raw(escFeed3)
	b.raw(gsCutPartial)
	return b.buf
}

func newBuilder(widthMM int) *escposBuilder {
	b := &escposBuilder{width: columnsFor(widthMM)}
	b.raw(escInit)
	b.raw(escCodepage858)
	return b
}

// BuildTicket renders a sale ticket as ESC/POS bytes for 58 mm (32 columns)
// or 80 mm (48 columns) paper. It mirrors the on-screen TicketPreview.
func BuildTicket(t model.TicketData, widthMM int) []byte {
	b := newBuilder(widthMM)

	if t.Status == "voided" {
		b.boldCentered("*** ANULADO ***")
	}
	if t.Business.Name != "" {
		b.boldCentered(strings.ToUpper(t.Business.Name))
	}
	if t.Business.Address != "" {
		b.centered(t.Business.Address)
	}
	if t.Business.NIT != "" {
		b.centered("NIT: " + t.Business.NIT)
	}
	if t.Business.TicketHeader != "" {
		b.centered(t.Business.TicketHeader)
	}
	b.divider()

	b.cols(fmt.Sprintf("Venta #%d", t.SaleID), t.CreatedAt.Format("02/01/2006"))
	b.cols("Método: "+methodLabel(t.PaymentMethod), t.CreatedAt.Format("03:04 PM"))

	if t.PaymentMethod == "mixed" && len(t.Payments) > 1 {
		for _, p := range t.Payments {
			b.cols(methodLabel(p.Method), formatMoney(p.Amount))
		}
	}
	if t.PaymentMethod == "credit" {
		b.boldCentered("*** CREDITO ***")
	}
	if t.Customer != nil {
		b.cols("Cliente: "+t.Customer.Name, "")
		if t.Customer.Phone != "" {
			b.cols("Tel: "+t.Customer.Phone, "")
		}
	}
	b.divider()

	for _, it := range t.Items {
		b.cols(it.ProductName, "")
		detail := ticketQty(it.Quantity, it.Unit) + " x " + formatMoney(it.UnitPrice)
		if it.Discount > 0 {
			detail += " -" + formatMoney(it.Discount)
		}
		b.cols(detail, formatMoney(it.Subtotal))
	}
	b.divider()

	if t.Discount > 0 {
		b.cols("Descuento", "-"+formatMoney(t.Discount))
	}
	if t.Tax > 0 {
		b.cols("IVA", formatMoney(t.Tax))
	}
	b.raw(escBoldOn)
	b.raw(gsDoubleHeight)
	b.cols("TOTAL", formatMoney(t.Total))
	b.raw(gsNormalSize)
	b.raw(escBoldOff)

	if t.PaymentMethod == "cash" && t.CashTendered != nil {
		b.cols("Recibido", formatMoney(*t.CashTendered))
		change := 0.0
		if t.ChangeAmount != nil {
			change = *t.ChangeAmount
		}
		b.raw(escBoldOn)
		b.cols("Cambio", formatMoney(change))
		b.raw(escBoldOff)
	}
	if t.Reference != "" {
		b.cols("Ref: "+t.Reference, "")
	}
	if t.Business.TicketFooter != "" {
		b.divider()
		b.centered(t.Business.TicketFooter)
	}
	b.divider()
	b.centered(t.CreatedAt.Format("02/01/2006") + " " + t.CreatedAt.Format("03:04 PM"))
	return b.finish()
}

// BuildTestTicket renders the short sample ticket used by "Imprimir prueba".
func BuildTestTicket(businessName, method string, widthMM int, now time.Time) []byte {
	b := newBuilder(widthMM)
	if businessName != "" {
		b.boldCentered(strings.ToUpper(businessName))
	}
	b.boldCentered("PRUEBA DE IMPRESION")
	b.divider()
	b.cols("Método", method)
	b.cols("Papel", fmt.Sprintf("%d mm", widthMM))
	b.cols("Fecha", now.Format("02/01/2006 03:04 PM"))
	b.divider()
	b.centered("Si puede leer este texto, la impresora está lista: ñ á é í ó ú ¿?")
	return b.finish()
}

// Transport delivers raw bytes to one physical printer.
type Transport interface {
	// Key identifies the printer; jobs with the same key are serialized.
	Key() string
	Send(ctx context.Context, data []byte) error
}

// NetworkTransport writes to a raw TCP printer (usually port 9100).
type NetworkTransport struct {
	Host string
	Port int
}

func (n NetworkTransport) Key() string {
	return net.JoinHostPort(n.Host, strconv.Itoa(n.Port))
}

func (n NetworkTransport) Send(ctx context.Context, data []byte) error {
	d := net.Dialer{Timeout: networkDialTimeout}
	conn, err := d.DialContext(ctx, "tcp", n.Key())
	if err != nil {
		return fmt.Errorf("connect to printer %s: %w", n.Key(), err)
	}
	defer conn.Close()
	if err := conn.SetWriteDeadline(time.Now().Add(networkWriteTimeout)); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("write to printer %s: %w", n.Key(), err)
	}
	return nil
}

// WindowsTransport prints raw bytes to a Windows print queue by name.
// Send is implemented per OS (printer_windows.go / printer_other.go).
type WindowsTransport struct {
	Name string
}

func (w WindowsTransport) Key() string { return w.Name }

// PrinterService serializes jobs per printer so bytes are never interleaved.
type PrinterService struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func NewPrinterService() *PrinterService {
	return &PrinterService{locks: map[string]*sync.Mutex{}}
}

func (s *PrinterService) lockFor(key string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.locks[key]
	if !ok {
		m = &sync.Mutex{}
		s.locks[key] = m
	}
	return m
}

// Print sends data through t, holding the mutex of t.Key() for the whole job.
func (s *PrinterService) Print(ctx context.Context, t Transport, data []byte) error {
	m := s.lockFor(t.Key())
	m.Lock()
	defer m.Unlock()
	return t.Send(ctx, data)
}
