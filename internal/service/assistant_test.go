package service

// Tests for AssistantService — contrato US-061 §4, T-1, T-3..T-11, T-15..T-17,
// T-24, T-25. Fixture pattern mirrors internal/handler/dashboard_test.go
// (setupTestDB) and internal/repository/metrics_test.go (metricsToday0):
// the legacy tests below anchor to real wall-clock "today" (the service clock
// defaults to time.Now); US-098 T-7 injects a fixed clock through svc.now.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/andreyo/pos/internal/database"
	"github.com/andreyo/pos/internal/repository"
)

// ─── fixture ──────────────────────────────────────────────────────────────

type assistantTestEnv struct {
	svc *AssistantService
	db  *database.DB
}

func newAssistantTestEnv(t *testing.T) *assistantTestEnv {
	t.Helper()
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	metricsRepo := repository.NewMetricsRepository(db.DB)
	reportRepo := repository.NewReportRepository(db.DB)
	expenseRepo := repository.NewExpenseRepository(db.DB)
	queryRepo := repository.NewAssistantQueryRepository(db.DB)

	return &assistantTestEnv{
		svc: NewAssistantService(metricsRepo, reportRepo, expenseRepo, queryRepo),
		db:  db,
	}
}

func (e *assistantTestEnv) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	conv := make([]any, len(args))
	for i, a := range args {
		if tm, ok := a.(time.Time); ok {
			conv[i] = tm.UTC().Format("2006-01-02 15:04:05") // UTC text, as CURRENT_TIMESTAMP stores it (US-098)
		} else {
			conv[i] = a
		}
	}
	if _, err := e.db.DB.ExecContext(context.Background(), q, conv...); err != nil {
		t.Fatalf("seed exec: %v\nSQL: %s", err, q)
	}
}

// today0 is 00:00 local time "today" (real wall clock) — same convention as
// internal/repository/metrics_test.go's metricsToday0.
func today0() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// startOfWeekLocal/startOfMonthLocal/startOfYearLocal duplicate the range
// starts internal/assistant/period.go computes, only so T-9's table can seed
// a sale that is guaranteed to fall inside each period's real [from, to)
// range regardless of what time of day the test happens to run (the engine
// itself is not reachable from this package to compute them directly).
func startOfWeekLocal(t time.Time) time.Time {
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	wd := int(day.Weekday())
	if wd == 0 {
		wd = 7
	}
	return day.AddDate(0, 0, -(wd - 1))
}

func startOfMonthLocal(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func startOfYearLocal(t time.Time) time.Time {
	return time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, t.Location())
}

const testUserID int64 = 1

// ─── T-1 (AC-1) ───────────────────────────────────────────────────────────

func TestAssistant_T1_SalesToday(t *testing.T) {
	env := newAssistantTestEnv(t)
	hoy0 := today0()
	env.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 28000, 28000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour))
	env.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 12000, 12000, 'completed', 'cash', ?)`, hoy0.Add(2*time.Hour))

	result, err := env.svc.Ask(context.Background(), testUserID, "cuanto vendi hoy")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !strings.Contains(result.Text, "$40.000") {
		t.Errorf("text = %q, want it to contain $40.000", result.Text)
	}
	if !strings.Contains(result.Text, "2") {
		t.Errorf("text = %q, want it to contain the count 2", result.Text)
	}
	if result.Intent != "sales" {
		t.Errorf("Intent = %q, want sales", result.Intent)
	}
}

// ─── T-3 (AC-2/AC-3) ──────────────────────────────────────────────────────

func TestAssistant_T3_TopProductsContrastLine(t *testing.T) {
	env := newAssistantTestEnv(t)
	hoy0 := today0()
	// Prod A: 90 u. @ 10.000, cost 8.900 -> margin 11%.
	env.exec(t, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (1, 'Prod A', 'A', 10000, 8900, 'unit', 1)`)
	// Prod B: 24 u. @ 10.000, cost 6.200 -> margin 38%.
	env.exec(t, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (2, 'Prod B', 'B', 10000, 6200, 'unit', 1)`)
	env.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1140000, 1140000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour))
	env.exec(t, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 1, 90, 10000, 900000)`)
	env.exec(t, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 2, 24, 10000, 240000)`)

	result, err := env.svc.Ask(context.Background(), testUserID, "que es lo que mas vendo")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !strings.Contains(result.Text, "Prod A") || !strings.Contains(result.Text, "Prod B") {
		t.Errorf("text = %q, want contrast line naming both Prod A and Prod B", result.Text)
	}

	// Second env: single product wins both rankings -> no contrast line.
	env2 := newAssistantTestEnv(t)
	env2.exec(t, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (1, 'Prod X', 'X', 10000, 8000, 'unit', 1)`)
	env2.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 100000, 100000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour))
	env2.exec(t, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 1, 10, 10000, 100000)`)

	result2, err := env2.svc.Ask(context.Background(), testUserID, "que es lo que mas vendo")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if strings.Contains(result2.Text, "pero lo que más plata te deja es") {
		t.Errorf("text = %q, want NO contrast line when the #1 by units and by margin coincide", result2.Text)
	}
}

// ─── T-4 (AC-4) ───────────────────────────────────────────────────────────

func TestAssistant_T4_UnconfiguredCostWarning(t *testing.T) {
	env := newAssistantTestEnv(t)
	hoy0 := today0()
	env.exec(t, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (1, 'Prod A', 'A', 10000, 5000, 'unit', 1)`)
	env.exec(t, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (2, 'Prod B', 'B', 6000, 0, 'unit', 1)`)
	env.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 16000, 16000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour))
	env.exec(t, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 1, 1, 10000, 10000)`)
	env.exec(t, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 2, 1, 6000, 6000)`)

	const want = "Ojo: 1 productos no tienen costo configurado"

	for _, q := range []string{"cuánto gané hoy", "cuál es mi margen hoy", "en qué gano más"} {
		result, err := env.svc.Ask(context.Background(), testUserID, q)
		if err != nil {
			t.Fatalf("Ask(%q): %v", q, err)
		}
		if !strings.Contains(result.Text, want) {
			t.Errorf("Ask(%q).Text = %q, want it to contain %q", q, result.Text, want)
		}
	}
}

// ─── T-5 (AC-5) ───────────────────────────────────────────────────────────

func TestAssistant_T5_NoDataNeverBareZero(t *testing.T) {
	env := newAssistantTestEnv(t)

	salesResult, err := env.svc.Ask(context.Background(), testUserID, "cuanto vendi hoy")
	if err != nil {
		t.Fatalf("Ask(sales): %v", err)
	}
	if salesResult.Text != "Todavía no hay ventas registradas hoy." {
		t.Errorf("sales text = %q, want the exact no-data phrase", salesResult.Text)
	}
	if strings.Contains(salesResult.Text, "$0") {
		t.Errorf("sales text = %q, must never contain a bare $0", salesResult.Text)
	}

	debtorsResult, err := env.svc.Ask(context.Background(), testUserID, "quien me debe")
	if err != nil {
		t.Fatalf("Ask(debtors): %v", err)
	}
	if debtorsResult.Text != "Nadie te debe plata ahora mismo. Cartera en cero." {
		t.Errorf("debtors text = %q, want the exact no-data phrase", debtorsResult.Text)
	}
}

// ─── T-6 (AC-6) ───────────────────────────────────────────────────────────

func TestAssistant_T6_UnknownReturnsThreeSuggestions(t *testing.T) {
	env := newAssistantTestEnv(t)

	result, err := env.svc.Ask(context.Background(), testUserID, "¿cómo hago un sándwich?")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if result.Intent != "unknown" {
		t.Errorf("Intent = %q, want unknown", result.Intent)
	}
	if len(result.Suggestions) != 3 {
		t.Errorf("len(Suggestions) = %d, want exactly 3: %v", len(result.Suggestions), result.Suggestions)
	}
}

// ─── T-7 (AC-7) ───────────────────────────────────────────────────────────

func TestAssistant_T7_AmbiguousOffersCandidates(t *testing.T) {
	env := newAssistantTestEnv(t)

	result, err := env.svc.Ask(context.Background(), testUserID, "cuanto bendi y gaine")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if result.Intent != "ambiguous" {
		t.Errorf("Intent = %q, want ambiguous", result.Intent)
	}
	if !strings.HasPrefix(result.Text, "¿Te refieres a") {
		t.Errorf("text = %q, want it to start with the clarifying question", result.Text)
	}
	if len(result.Suggestions) != 2 {
		t.Errorf("len(Suggestions) = %d, want exactly 2 (SALES/PROFIT): %v", len(result.Suggestions), result.Suggestions)
	}
	// US-062 (AC-15a): first-person chips the engine resolves on their own.
	wantChips := []string{"cuánto vendí", "cuánto gané"}
	if len(result.Suggestions) == len(wantChips) {
		for i, w := range wantChips {
			if result.Suggestions[i] != w {
				t.Errorf("Suggestions[%d] = %q, want %q", i, result.Suggestions[i], w)
			}
		}
	}
	// The clarifying text keeps the 2nd-person wording (contract AC-15a).
	if result.Text != "¿Te refieres a cuánto vendiste o a cuánto ganaste?" {
		t.Errorf("text = %q, want the unchanged clarifying question", result.Text)
	}
}

// ─── T-8 (AC-8) ───────────────────────────────────────────────────────────

func TestAssistant_T8_CompareShowsSignedDrop(t *testing.T) {
	env := newAssistantTestEnv(t)
	hoy0 := today0()
	env.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 28000, 28000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour))
	env.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 12000, 12000, 'completed', 'cash', ?)`, hoy0.Add(2*time.Hour))
	env.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (3, 1, 50000, 50000, 'completed', 'cash', ?)`, hoy0.Add(-12*time.Hour))

	result, err := env.svc.Ask(context.Background(), testUserID, "cuanto vendi hoy comparado con ayer")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !strings.Contains(result.Text, "$40.000") || !strings.Contains(result.Text, "$50.000") {
		t.Errorf("text = %q, want both figures 40.000 and 50.000", result.Text)
	}
	if !strings.Contains(result.Text, "-20%") {
		t.Errorf("text = %q, want the signed drop -20%%", result.Text)
	}
}

func TestAssistant_T8_CompareShowsSignedRise(t *testing.T) {
	env := newAssistantTestEnv(t)
	hoy0 := today0()
	env.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 60000, 60000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour))
	env.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 50000, 50000, 'completed', 'cash', ?)`, hoy0.Add(-12*time.Hour))

	result, err := env.svc.Ask(context.Background(), testUserID, "cuanto vendi hoy comparado con ayer")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	// The sign must be explicit even on the rise (GATE 1 decision): "+20%", never a bare "20%".
	if !strings.Contains(result.Text, "+20%") {
		t.Errorf("text = %q, want the explicit sign +20%%", result.Text)
	}
}

// ─── T-9 (AC-9) ───────────────────────────────────────────────────────────

func TestAssistant_T9_PeriodLabelNamedInText(t *testing.T) {
	now := time.Now()
	hoy0 := today0()
	monday := startOfWeekLocal(now)
	monthStart := startOfMonthLocal(now)
	yearStart := startOfYearLocal(now)

	cases := []struct {
		phrase    string
		wantLabel string
		saleAt    time.Time
	}{
		{"cuánto vendí hoy", "Hoy", hoy0},
		{"cuánto vendí ayer", "Ayer", hoy0.Add(-time.Hour)},
		{"cuánto vendí esta semana", "Esta semana", monday},
		{"cuánto vendí la semana pasada", "Semana pasada", monday.Add(-3 * 24 * time.Hour)},
		{"cuánto vendí este mes", "Este mes", monthStart},
		{"cuánto vendí el mes pasado", "Mes pasado", monthStart.AddDate(0, 0, -15)},
		{"cuánto vendí este año", "Este año", yearStart},
	}

	for _, c := range cases {
		env := newAssistantTestEnv(t)
		env.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 10000, 10000, 'completed', 'cash', ?)`, c.saleAt)

		result, err := env.svc.Ask(context.Background(), testUserID, c.phrase)
		if err != nil {
			t.Fatalf("Ask(%q): %v", c.phrase, err)
		}
		if !strings.Contains(result.Text, c.wantLabel) {
			t.Errorf("Ask(%q).Text = %q, want it to contain the label %q", c.phrase, result.Text, c.wantLabel)
		}
	}
}

// ─── T-10 (AC-10) ─────────────────────────────────────────────────────────

func TestAssistant_T10_UnresolvedPeriodAsksForDate(t *testing.T) {
	env := newAssistantTestEnv(t)
	hoy0 := today0()
	env.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 40000, 40000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour))
	env.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 50000, 50000, 'completed', 'cash', ?)`, hoy0.Add(-12*time.Hour))

	result, err := env.svc.Ask(context.Background(), testUserID, "cuanto vendi el juebes")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if strings.Contains(result.Text, "40.000") || strings.Contains(result.Text, "50.000") {
		t.Errorf("text = %q, must NOT leak any figure when the period is unresolved", result.Text)
	}
	// US-062 (AC-15b): "cuanto vendi el juebes" is a sales query, so the chips
	// are full sales phrases the engine resolves on their own.
	want := []string{"cuánto vendí hoy", "cuánto vendí ayer", "cuánto vendí esta semana", "cuánto vendí este mes"}
	if len(result.Suggestions) != len(want) {
		t.Fatalf("Suggestions = %v, want %v", result.Suggestions, want)
	}
	for i, w := range want {
		if result.Suggestions[i] != w {
			t.Errorf("Suggestions[%d] = %q, want %q", i, result.Suggestions[i], w)
		}
	}
}

// ─── T-11 (AC-11) ─────────────────────────────────────────────────────────

func TestAssistant_T11_FormatCOP(t *testing.T) {
	if got := formatCOP(1234567.4); got != "$1.234.567" {
		t.Errorf("formatCOP(1234567.4) = %q, want $1.234.567", got)
	}
	if got := formatCOP(-5000); got != "-$5.000" {
		t.Errorf("formatCOP(-5000) = %q, want -$5.000 (documented sign decision)", got)
	}
}

// ─── T-15/T-16 (AC-15/AC-16) ──────────────────────────────────────────────

// bitacoraRow reads back one row from assistant_queries for assertion.
type bitacoraRow struct {
	userID     int64
	query      string
	intent     string
	tier       string
	confidence float64
}

func (e *assistantTestEnv) lastBitacoraRow(t *testing.T) bitacoraRow {
	t.Helper()
	var row bitacoraRow
	err := e.db.DB.QueryRowContext(context.Background(),
		`SELECT user_id, query, intent, tier, confidence FROM assistant_queries ORDER BY id DESC LIMIT 1`,
	).Scan(&row.userID, &row.query, &row.intent, &row.tier, &row.confidence)
	if err != nil {
		t.Fatalf("read bitacora row: %v", err)
	}
	return row
}

func TestAssistant_T15_BitacoraLogsUnknownQueryWithoutFigures(t *testing.T) {
	env := newAssistantTestEnv(t)
	const q = "¿cómo hago un sándwich?"

	if _, err := env.svc.Ask(context.Background(), testUserID, q); err != nil {
		t.Fatalf("Ask: %v", err)
	}

	row := env.lastBitacoraRow(t)
	if row.userID != testUserID {
		t.Errorf("user_id = %d, want %d", row.userID, testUserID)
	}
	if row.query != q {
		t.Errorf("query = %q, want %q", row.query, q)
	}
	if row.intent != "unknown" {
		t.Errorf("intent = %q, want unknown", row.intent)
	}
	if row.tier != "none" {
		t.Errorf("tier = %q, want none (never empty string)", row.tier)
	}
}

func TestAssistant_T16_TierMapping(t *testing.T) {
	cases := []struct {
		query    string
		wantTier string
	}{
		{"cuánto vendí hoy", "regex"},       // exact T1 phrase
		{"cuanto vendi ayer", "fuzzy"},      // no exact T1 phrase, resolves via T2
		{"¿cómo hago un sándwich?", "none"}, // no match at all
	}
	for _, c := range cases {
		env := newAssistantTestEnv(t)
		if _, err := env.svc.Ask(context.Background(), testUserID, c.query); err != nil {
			t.Fatalf("Ask(%q): %v", c.query, err)
		}
		row := env.lastBitacoraRow(t)
		if row.tier != c.wantTier {
			t.Errorf("Ask(%q) bitacora tier = %q, want %q", c.query, row.tier, c.wantTier)
		}
	}
}

// ─── T-17 (AC-17) ─────────────────────────────────────────────────────────

// erroringQueryRepo simulates a bitácora insert that always fails, without
// tearing down the whole in-memory DB (which would also break the real
// business queries the answer itself depends on).
type erroringQueryRepo struct{}

func (erroringQueryRepo) Create(ctx context.Context, userID int64, query, intent, tier string, confidence float64) error {
	return errors.New("simulated bitacora failure")
}

func TestAssistant_T17_BitacoraFailureNeverBlocksTheAnswer(t *testing.T) {
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	hoy0 := today0()
	if _, err := db.DB.ExecContext(context.Background(),
		`INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 40000, 40000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour).UTC().Format("2006-01-02 15:04:05"),
	); err != nil {
		t.Fatalf("seed exec: %v", err)
	}

	svc := NewAssistantService(
		repository.NewMetricsRepository(db.DB),
		repository.NewReportRepository(db.DB),
		repository.NewExpenseRepository(db.DB),
		erroringQueryRepo{},
	)

	result, err := svc.Ask(context.Background(), testUserID, "cuanto vendi hoy")
	if err != nil {
		t.Fatalf("Ask returned an error even though only the bitacora insert failed: %v", err)
	}
	if !strings.Contains(result.Text, "$40.000") {
		t.Errorf("text = %q, want the computed answer despite the bitacora failure", result.Text)
	}
}

// ─── T-24 (AC-21, service) ────────────────────────────────────────────────

func TestAssistant_T24_TopProductsOrdersByUnits(t *testing.T) {
	env := newAssistantTestEnv(t)
	hoy0 := today0()
	// A: 50 u. / $100.000 revenue. B: 80 u. / $40.000 revenue. C: 10 u. / $200.000 revenue.
	env.exec(t, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (1, 'A', 'A', 2000, 1000, 'unit', 1)`)
	env.exec(t, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (2, 'B', 'B', 500, 300, 'unit', 1)`)
	env.exec(t, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (3, 'C', 'C', 20000, 15000, 'unit', 1)`)
	env.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 340000, 340000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour))
	env.exec(t, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 1, 50, 2000, 100000)`)
	env.exec(t, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 2, 80, 500, 40000)`)
	env.exec(t, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 3, 10, 20000, 200000)`)

	result, err := env.svc.Ask(context.Background(), testUserID, "que es lo que mas vendo")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	idxB := strings.Index(result.Text, "B")
	idxA := strings.Index(result.Text, "A")
	idxC := strings.Index(result.Text, "C")
	if idxB < 0 || idxA < 0 || idxC < 0 {
		t.Fatalf("text = %q, want it to list A, B and C", result.Text)
	}
	if !(idxB < idxA && idxA < idxC) {
		t.Errorf("text = %q, want order B (80u), A (50u), C (10u) by units — got B@%d A@%d C@%d", result.Text, idxB, idxA, idxC)
	}
}

// ─── T-25 (AC-22) ─────────────────────────────────────────────────────────

func TestAssistant_T25_PaymentMixClarifiesNequi(t *testing.T) {
	env := newAssistantTestEnv(t)
	hoy0 := today0()
	env.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 30000, 30000, 'completed', 'transfer', ?)`, hoy0.Add(time.Hour))
	env.exec(t, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'transfer', 30000)`)

	result, err := env.svc.Ask(context.Background(), testUserID, "cuanto me entro por nequi")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !strings.Contains(result.Text, "Nequi") || !strings.Contains(result.Text, "Daviplata") {
		t.Errorf("text = %q, want the explicit Nequi/Daviplata clarification", result.Text)
	}
}

// ─── T-12 (US-062 AC-15a, AC-15c) ─────────────────────────────────────────

func TestAssistant_T12_AmbiguousChipsRoundTrip(t *testing.T) {
	env := newAssistantTestEnv(t)

	result, err := env.svc.Ask(context.Background(), testUserID, "cuanto bendi y gaine")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if result.Intent != "ambiguous" {
		t.Fatalf("Intent = %q, want ambiguous", result.Intent)
	}
	wantChips := []string{"cuánto vendí", "cuánto gané"}
	if len(result.Suggestions) != len(wantChips) {
		t.Fatalf("Suggestions = %v, want exactly %v", result.Suggestions, wantChips)
	}
	for i, w := range wantChips {
		if result.Suggestions[i] != w {
			t.Errorf("Suggestions[%d] = %q, want %q", i, result.Suggestions[i], w)
		}
	}

	// Round trip: each chip resolves to its own intent, tier regex, label Hoy.
	wantIntents := []string{"sales", "profit"}
	for i, chip := range result.Suggestions {
		got, err := env.svc.Ask(context.Background(), testUserID, chip)
		if err != nil {
			t.Fatalf("Ask(%q): %v", chip, err)
		}
		if got.Intent != wantIntents[i] {
			t.Errorf("Ask(%q).Intent = %q, want %q", chip, got.Intent, wantIntents[i])
		}
		if got.Tier != "regex" {
			t.Errorf("Ask(%q).Tier = %q, want regex", chip, got.Tier)
		}
		if got.Period.Label != "Hoy" {
			t.Errorf("Ask(%q).Period.Label = %q, want Hoy", chip, got.Period.Label)
		}
	}

	// Table A (contract §2.4): all 11 intents, chip -> same intent.
	tableA := []struct {
		chip   string
		intent string
	}{
		{"cuánto vendí", "sales"},
		{"cuánto gané", "profit"},
		{"cuál es mi margen", "margin"},
		{"qué es lo que más vendo", "top_products"},
		{"en qué gano más", "top_margin"},
		{"qué se está acabando", "low_stock"},
		{"quién me debe", "debtors"},
		{"cuánto he gastado", "expenses"},
		{"cuánto me entró por nequi", "payment_mix"},
		{"cuánta plata hay en caja", "cash"},
		{"cuánto fié hoy", "credit_flow"},
	}
	for _, c := range tableA {
		got, err := env.svc.Ask(context.Background(), testUserID, c.chip)
		if err != nil {
			t.Fatalf("Ask(%q): %v", c.chip, err)
		}
		if got.Intent != c.intent {
			t.Errorf("Ask(%q).Intent = %q, want %q", c.chip, got.Intent, c.intent)
		}
	}
}

// ─── T-13 (US-062 AC-15b, AC-15c) ─────────────────────────────────────────

func TestAssistant_T13_UnresolvedChipsRoundTrip(t *testing.T) {
	env := newAssistantTestEnv(t)
	hoy0 := today0()
	// Same seed as T10: $40.000 today, $50.000 yesterday.
	env.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 40000, 40000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour))
	env.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 50000, 50000, 'completed', 'cash', ?)`, hoy0.Add(-12*time.Hour))

	assertChips := func(query string, want []string) {
		t.Helper()
		got, err := env.svc.Ask(context.Background(), testUserID, query)
		if err != nil {
			t.Fatalf("Ask(%q): %v", query, err)
		}
		if len(got.Suggestions) != len(want) {
			t.Fatalf("Ask(%q).Suggestions = %v, want exactly %v", query, got.Suggestions, want)
		}
		for i, w := range want {
			if got.Suggestions[i] != w {
				t.Errorf("Ask(%q).Suggestions[%d] = %q, want %q", query, i, got.Suggestions[i], w)
			}
		}
	}

	salesChips := []string{"cuánto vendí hoy", "cuánto vendí ayer", "cuánto vendí esta semana", "cuánto vendí este mes"}
	assertChips("cuanto vendi el juebes", salesChips)
	assertChips("cuanto gane el juebes", []string{"cuánto gané hoy", "cuánto gané ayer", "cuánto gané esta semana"})
	assertChips("quien me debe el juebes", []string{"quién me debe"})

	// Round trip of the sales chips with real figures.
	hoy, err := env.svc.Ask(context.Background(), testUserID, salesChips[0])
	if err != nil {
		t.Fatalf("Ask(hoy): %v", err)
	}
	if hoy.Period.Label != "Hoy" || !strings.Contains(hoy.Text, "$40.000") {
		t.Errorf("hoy: label=%q text=%q, want label Hoy and $40.000", hoy.Period.Label, hoy.Text)
	}
	ayer, err := env.svc.Ask(context.Background(), testUserID, salesChips[1])
	if err != nil {
		t.Fatalf("Ask(ayer): %v", err)
	}
	if ayer.Period.Label != "Ayer" || !strings.Contains(ayer.Text, "$50.000") {
		t.Errorf("ayer: label=%q text=%q, want label Ayer and $50.000", ayer.Period.Label, ayer.Text)
	}
	for i, wantLabel := range map[int]string{2: "Esta semana", 3: "Este mes"} {
		got, err := env.svc.Ask(context.Background(), testUserID, salesChips[i])
		if err != nil {
			t.Fatalf("Ask(%q): %v", salesChips[i], err)
		}
		if got.Intent != "sales" || got.Period.Label != wantLabel {
			t.Errorf("Ask(%q): intent=%q label=%q, want sales / %q", salesChips[i], got.Intent, got.Period.Label, wantLabel)
		}
	}

	// Table B (contract §2.4): every chip of every intent resolves to the same
	// intent, with a period label and without asking for a date again.
	tableB := []struct {
		intent string
		chips  []string
	}{
		{"sales", salesChips},
		{"profit", []string{"cuánto gané hoy", "cuánto gané ayer", "cuánto gané esta semana"}},
		{"margin", []string{"cuál es mi margen hoy", "cuál es mi margen ayer", "cuál es mi margen esta semana", "cuál es mi margen este mes"}},
		{"top_products", []string{"cuál es mi producto estrella hoy", "cuál es mi producto estrella ayer", "cuál es mi producto estrella esta semana", "cuál es mi producto estrella este mes"}},
		{"top_margin", []string{"en qué gano más hoy", "en qué gano más ayer"}},
		{"low_stock", []string{"qué se está acabando"}},
		{"debtors", []string{"quién me debe"}},
		{"expenses", []string{"cuánto he gastado hoy", "cuánto he gastado ayer", "cuánto he gastado esta semana", "cuánto he gastado este mes"}},
		{"payment_mix", []string{"cuánto me entró por nequi hoy", "cuánto me entró por nequi ayer", "cuánto me entró por nequi esta semana", "cuánto me entró por nequi este mes"}},
		{"cash", []string{"cuánto efectivo tengo hoy", "cuánto efectivo tengo ayer", "cuánto efectivo tengo esta semana", "cuánto efectivo tengo este mes"}},
		{"credit_flow", []string{"cuánto fié hoy", "cuánto fié ayer", "cuánto fié esta semana"}},
	}
	for _, row := range tableB {
		for _, chip := range row.chips {
			got, err := env.svc.Ask(context.Background(), testUserID, chip)
			if err != nil {
				t.Fatalf("Ask(%q): %v", chip, err)
			}
			if got.Intent != row.intent {
				t.Errorf("Ask(%q).Intent = %q, want %q", chip, got.Intent, row.intent)
			}
			if got.Period.Label == "" {
				t.Errorf("Ask(%q).Period.Label is empty, want a resolved period", chip)
			}
			if strings.Contains(got.Text, "De qué fecha") {
				t.Errorf("Ask(%q).Text = %q, must not ask for a date again", chip, got.Text)
			}
		}
	}

	// Each unresolved-period query of table B must itself emit that intent's
	// own row (order and content), not just the sales list.
	for _, q := range []struct {
		query string
		want  []string
	}{
		{"cuanto gane el juebes", tableB[1].chips},
		{"quien me debe el juebes", tableB[6].chips},
	} {
		assertChips(q.query, q.want)
	}
}

// ─── US-098: day windows and stored UTC timestamps agree ────────────────────

// T-7 (AC-6): clock 2026-11-09 20:30 Bogota. Stored text is UTC, so the
// 20:18:41 sale (2026-11-10 01:18:41 UTC) belongs to "hoy".
//
// Seed (contract §3, literal UTC text, all completed cash sales):
// S0 2026-11-09 04:59:59 $5.000, S5 2026-11-09 05:00:00 $7.000,
// S1 2026-11-09 23:59:00 $10.000, S2 2026-11-10 01:18:41 $50.000,
// S3 2026-11-10 04:59:59 $20.000, S4 2026-11-10 05:00:00 $3.000.
func TestAssistant_T7_DayWindowsAtEvening(t *testing.T) {
	bogota := time.FixedZone("COT", -5*3600)
	env := newAssistantTestEnv(t)
	env.svc.now = func() time.Time { return time.Date(2026, 11, 9, 20, 30, 0, 0, bogota) }
	for _, s := range []struct {
		id    int
		total int
		at    string
	}{
		{10, 5000, "2026-11-09 04:59:59"},
		{15, 7000, "2026-11-09 05:00:00"},
		{11, 10000, "2026-11-09 23:59:00"},
		{12, 50000, "2026-11-10 01:18:41"},
		{13, 20000, "2026-11-10 04:59:59"},
		{14, 3000, "2026-11-10 05:00:00"},
	} {
		env.exec(t, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (?, 1, ?, ?, 'completed', 'cash', ?)`,
			s.id, s.total, s.total, s.at)
	}

	ask := func(phrase string) salesData {
		t.Helper()
		result, err := env.svc.Ask(context.Background(), testUserID, phrase)
		if err != nil {
			t.Fatalf("Ask(%q): %v", phrase, err)
		}
		data, ok := result.Data.(salesData)
		if !ok {
			t.Fatalf("Ask(%q).Data = %#v, want salesData (text %q)", phrase, result.Data, result.Text)
		}
		return data
	}

	// hoy: S5 + S1 + S2 + S3 = 87.000 / 4 sales.
	if d := ask("cuánto vendí hoy"); d.Total != 87000 || d.Count != 4 {
		t.Errorf("hoy = %.0f / %d, want 87000 / 4", d.Total, d.Count)
	}
	// ayer: S0 = 5.000 / 1.
	if d := ask("cuánto vendí ayer"); d.Total != 5000 || d.Count != 1 {
		t.Errorf("ayer = %.0f / %d, want 5000 / 1", d.Total, d.Count)
	}
	// este mes: [Nov 1 00:00, now 20:30) = S0 + S5 + S1 + S2 = 72.000 / 4 (S3 and S4
	// are later than the clock). Without S2 it would be 22.000 / 3.
	if d := ask("cuánto vendí este mes"); d.Total != 72000 || d.Count != 4 {
		t.Errorf("este mes = %.0f / %d, want 72000 / 4 (S0 + S5 + S1 + S2; S2 included)", d.Total, d.Count)
	}
}
