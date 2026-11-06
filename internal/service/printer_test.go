package service

// US-080 contract section 4, T-1..T-6 and T-12 (native ESC/POS printing).
//
// Signatures assumed from contract section 5 (the contract does not spell them out):
//   BuildTicket(t model.TicketData, widthMM int) []byte
//   type Transport interface { Key() string; Send(ctx context.Context, data []byte) error }
//   NewPrinterService() *PrinterService ; (*PrinterService).Print(ctx, Transport, []byte) error

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/andreyo/pos/internal/model"
)

func f64(v float64) *float64 { return &v }

// textLines strips ESC/GS command sequences and splits the remaining bytes on
// LF. Each returned string keeps one char per byte (CP858 is single-byte).
func textLines(b []byte) []string {
	var out bytes.Buffer
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case 0x1B:
			if i+1 < len(b) {
				if b[i+1] == '@' {
					i++
				} else {
					i += 2 // ESC <cmd> <param>
				}
			}
		case 0x1D:
			i += 2 // GS <cmd> <param>
		default:
			out.WriteByte(b[i])
		}
	}
	return strings.Split(out.String(), "\n")
}

func lineWith(lines []string, sub string) (string, bool) {
	for _, l := range lines {
		if strings.Contains(l, sub) {
			return l, true
		}
	}
	return "", false
}

func firstNonEmpty(lines []string) string {
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			return strings.TrimSpace(l)
		}
	}
	return ""
}

func sampleSale() model.TicketData {
	return model.TicketData{
		SaleID:        1,
		CreatedAt:     time.Date(2026, 11, 5, 10, 30, 0, 0, time.Local),
		PaymentMethod: "cash",
		CashTendered:  f64(20000),
		ChangeAmount:  f64(9800),
		Status:        "completed",
		Subtotal:      10200,
		Total:         10200,
		Items: []model.TicketItem{
			{ProductName: "Gaseosa 1.5L", Quantity: 2, UnitPrice: 4500, Subtotal: 9000, Unit: "unit"},
			{ProductName: "Pan", Quantity: 1, UnitPrice: 1200, Subtotal: 1200, Unit: "unit"},
		},
		Payments: []model.Payment{{Method: "cash", Amount: 10200}},
		Business: model.TicketBusiness{Name: "Tienda Uno", Address: "Calle 1 # 2-3", NIT: "900123456-7", TicketFooter: "Gracias por su compra"},
	}
}

func totalLine(t *testing.T, lines []string) string {
	t.Helper()
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "TOTAL") {
			return l
		}
	}
	t.Fatalf("no line starting with TOTAL in %q", lines)
	return ""
}

func TestPrinterBuildTicket_T1_Layout58mm(t *testing.T) {
	out := BuildTicket(sampleSale(), 58)

	if !bytes.HasPrefix(out, []byte{0x1B, 0x40, 0x1B, 0x74, 0x13}) {
		t.Errorf("output must start with ESC @ ESC t 19, got % X", out[:min(5, len(out))])
	}
	if !bytes.HasSuffix(out, []byte{0x1B, 0x64, 0x03, 0x1D, 0x56, 0x01}) {
		t.Errorf("output must end with feed 3 (1B 64 03) then partial cut (1D 56 01)")
	}

	lines := textLines(out)
	for _, l := range lines {
		if len(l) > 32 {
			t.Errorf("line longer than 32 columns (%d): %q", len(l), l)
		}
	}
	tl := totalLine(t, lines)
	if !strings.HasSuffix(strings.TrimRight(tl, " "), "$10.200") {
		t.Errorf("TOTAL line = %q, want it to carry $10.200", tl)
	}
	rec, ok := lineWith(lines, "Recibido")
	if !ok || !strings.Contains(rec, "$20.000") {
		t.Errorf("Recibido line = %q (found %v), want $20.000", rec, ok)
	}
	cam, ok := lineWith(lines, "Cambio")
	if !ok || !strings.Contains(cam, "$9.800") {
		t.Errorf("Cambio line = %q (found %v), want $9.800", cam, ok)
	}
	if l, ok := lineWith(lines, "$9.000"); !ok || !strings.HasSuffix(l, "$9.000") {
		t.Errorf("item subtotal $9.000 must be right-aligned, got %q (found %v)", l, ok)
	}
	if _, ok := lineWith(lines, "$1.200"); !ok {
		t.Errorf("Pan line with $1.200 missing")
	}
	if _, ok := lineWith(lines, "NIT: 900123456-7"); !ok {
		t.Errorf("NIT line missing")
	}
}

func TestPrinterBuildTicket_T2_Layout80mm(t *testing.T) {
	lines := textLines(BuildTicket(sampleSale(), 80))
	for _, l := range lines {
		if len(l) > 48 {
			t.Errorf("line longer than 48 columns (%d): %q", len(l), l)
		}
	}
	tl := totalLine(t, lines)
	if len(tl) != 48 || !strings.HasSuffix(tl, "$10.200") {
		t.Errorf("TOTAL line = %q (len %d), want right-aligned to column 48 ending in $10.200", tl, len(tl))
	}
}

func TestPrinterBuildTicket_T3_LongNameTruncated(t *testing.T) {
	name := strings.Repeat("Abcdefghij", 6) // 60 chars
	tk := sampleSale()
	tk.Items = []model.TicketItem{{ProductName: name, Quantity: 3, UnitPrice: 3000, Subtotal: 9000, Unit: "unit"}}
	tk.Subtotal, tk.Total = 9000, 9000
	tk.Payments = []model.Payment{{Method: "cash", Amount: 9000}}

	lines := textLines(BuildTicket(tk, 58))
	for _, l := range lines {
		if len(l) > 32 {
			t.Errorf("line longer than 32 columns (%d): %q", len(l), l)
		}
		if strings.Contains(l, name) {
			t.Errorf("full 60-char name must be truncated, found in %q", l)
		}
	}
	if _, ok := lineWith(lines, "Abcdefghij"); !ok {
		t.Errorf("truncated name prefix missing")
	}
	var sub string
	for _, l := range lines {
		if strings.Contains(l, "$9.000") && !strings.HasPrefix(strings.TrimSpace(l), "TOTAL") {
			sub = l
		}
	}
	if len(sub) != 32 || !strings.HasSuffix(sub, "$9.000") {
		t.Errorf("subtotal line = %q (len %d), want length 32 ending in $9.000", sub, len(sub))
	}
}

func TestPrinterBuildTicket_T4_CP858Encoding(t *testing.T) {
	tk := sampleSale()
	tk.Business.Address = "Calle 5 Aquí"
	tk.Items = []model.TicketItem{{ProductName: "Café Crédito ñandú ¿Qué?", Quantity: 1, UnitPrice: 1234567, Subtotal: 1234567, Unit: "unit"}}
	tk.Subtotal, tk.Total = 1234567, 1234567
	tk.Payments = []model.Payment{{Method: "cash", Amount: 1234567}}
	tk.Business.TicketFooter = "Gracias 漢 \U0001F600"

	out := BuildTicket(tk, 80)

	if !bytes.Contains(out, []byte("Caf\x82 Cr\x82dito \xa4and\xa3 \xa8Qu\x82?")) {
		t.Errorf("accented item name not CP858-encoded: % X", out)
	}
	if !bytes.Contains(out, []byte("Aqu\xa1")) {
		t.Errorf("i-acute must encode as A1")
	}
	if !bytes.Contains(out, []byte("$1.234.567")) {
		t.Errorf("amount must be formatted $1.234.567")
	}
	if !bytes.Contains(out, []byte("Gracias ? ?")) {
		t.Errorf("unmappable runes (CJK, emoji) must each become one '?', got footer in % X", out)
	}

	tk.Business.TicketFooter = "áéíóúüñÁÉÍÓÚÜÑ¿¡€"
	want := []byte{0xA0, 0x82, 0xA1, 0xA2, 0xA3, 0x81, 0xA4, 0xB5, 0x90, 0xD6, 0xE0, 0xE9, 0x9A, 0xA5, 0xA8, 0xAD, 0xD5}
	if !bytes.Contains(BuildTicket(tk, 80), want) {
		t.Errorf("full CP858 table mismatch, want sequence % X", want)
	}
}

func TestPrinterBuildTicket_T5_VoidedAndCredit(t *testing.T) {
	v := sampleSale()
	v.Status = "voided"
	if got := firstNonEmpty(textLines(BuildTicket(v, 58))); got != "*** ANULADO ***" {
		t.Errorf("first text line of a voided sale = %q, want *** ANULADO ***", got)
	}

	c := sampleSale()
	c.PaymentMethod = "credit"
	c.CashTendered, c.ChangeAmount = nil, nil
	c.Payments = []model.Payment{{Method: "credit", Amount: 10200}}
	c.Customer = &model.Customer{Name: "Doña Rosa", Phone: "3001234567"}
	out := BuildTicket(c, 58)
	if !bytes.Contains(out, []byte("*** CREDITO ***")) {
		t.Errorf("credit sale must print *** CREDITO ***")
	}
	if !bytes.Contains(out, []byte("Do\xa4a Rosa")) {
		t.Errorf("customer name must be CP858 encoded as Do\\xA4a Rosa")
	}
	if !bytes.Contains(out, []byte("3001234567")) {
		t.Errorf("customer phone missing")
	}
	if bytes.Contains(BuildTicket(sampleSale(), 58), []byte("ANULADO")) {
		t.Errorf("completed sale must not print ANULADO")
	}
}

func TestPrinterBuildTicket_T6_DiscountAndTaxOnlyWhenPositive(t *testing.T) {
	tk := sampleSale()
	tk.Discount, tk.Tax = 1000, 1900
	lines := textLines(BuildTicket(tk, 58))
	d, ok := lineWith(lines, "Descuento")
	if !ok || !strings.HasSuffix(strings.TrimRight(d, " "), "-$1.000") {
		t.Errorf("Descuento line = %q (found %v), want -$1.000", d, ok)
	}
	iva, ok := lineWith(lines, "IVA")
	if !ok || !strings.HasSuffix(strings.TrimRight(iva, " "), "$1.900") {
		t.Errorf("IVA line = %q (found %v), want $1.900", iva, ok)
	}

	zero := textLines(BuildTicket(sampleSale(), 58))
	if l, ok := lineWith(zero, "Descuento"); ok {
		t.Errorf("Descuento must be absent when 0, got %q", l)
	}
	if l, ok := lineWith(zero, "IVA"); ok {
		t.Errorf("IVA must be absent when 0, got %q", l)
	}
}

// --- T-12: per-printer serialization (AC-9) ---

type tracker struct {
	inflight    int32
	maxInflight int32
	mu          sync.Mutex
	completed   [][]byte
}

type fakeTransport struct {
	key string
	tr  *tracker
	// hold, if set, is called while the job is "on the wire".
	hold func()
}

func (f *fakeTransport) Key() string { return f.key }

func (f *fakeTransport) Send(ctx context.Context, data []byte) error {
	n := atomic.AddInt32(&f.tr.inflight, 1)
	for {
		m := atomic.LoadInt32(&f.tr.maxInflight)
		if n <= m || atomic.CompareAndSwapInt32(&f.tr.maxInflight, m, n) {
			break
		}
	}
	if f.hold != nil {
		f.hold()
	} else {
		time.Sleep(20 * time.Millisecond)
	}
	f.tr.mu.Lock()
	f.tr.completed = append(f.tr.completed, append([]byte(nil), data...))
	f.tr.mu.Unlock()
	atomic.AddInt32(&f.tr.inflight, -1)
	return nil
}

func TestPrinterService_T12_SamePrinterJobsAreSerialized(t *testing.T) {
	svc := NewPrinterService()
	tr := &tracker{}
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			data := bytes.Repeat([]byte{byte('A' + i)}, 16)
			if err := svc.Print(context.Background(), &fakeTransport{key: "192.168.1.50:9100", tr: tr}, data); err != nil {
				t.Errorf("job %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	if got := atomic.LoadInt32(&tr.maxInflight); got != 1 {
		t.Errorf("max in-flight for the same printer = %d, want 1", got)
	}
	if len(tr.completed) != 5 {
		t.Fatalf("completed jobs = %d, want 5", len(tr.completed))
	}
	for _, d := range tr.completed {
		if len(d) != 16 || len(bytes.Trim(d, string(d[:1]))) != 0 {
			t.Errorf("job bytes interleaved or truncated: %q", d)
		}
	}
}

func TestPrinterService_T12_DifferentPrintersRunConcurrently(t *testing.T) {
	svc := NewPrinterService()
	tr := &tracker{}
	var entered int32
	// Each job waits (up to 2 s) until the other one is also on the wire.
	hold := func() {
		atomic.AddInt32(&entered, 1)
		deadline := time.Now().Add(2 * time.Second)
		for atomic.LoadInt32(&entered) < 2 && time.Now().Before(deadline) {
			time.Sleep(2 * time.Millisecond)
		}
	}
	var wg sync.WaitGroup
	for _, key := range []string{"192.168.1.50:9100", "POS-58"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			_ = svc.Print(context.Background(), &fakeTransport{key: key, tr: tr, hold: hold}, []byte("x"))
		}(key)
	}
	wg.Wait()
	if got := atomic.LoadInt32(&tr.maxInflight); got != 2 {
		t.Errorf("max in-flight across two printers = %d, want 2", got)
	}
}
