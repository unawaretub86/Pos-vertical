package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/andreyo/pos/internal/assistant"
	"github.com/andreyo/pos/internal/repository"
)

// ─── Fetch/display tuning ────────────────────────────────────────────────────

// assistantDisplayLimit is how many rows a list-shaped answer shows before
// summarizing the rest as "y N más" (sprint DoD, chat widget cannot render
// long lists).
const assistantDisplayLimit = 5

// assistantFetchLimit is what the service asks the repository for when it
// needs "all" rows to compute an accurate remainder count. Some repository
// methods (LowStock, TopByMargin, TopSellers) take LIMIT literally and have
// no "0 = unlimited" convention (unlike MetricsRepository.Debtors), so a
// generous constant is used instead.
const assistantFetchLimit = 1000

// ─── Response DTOs (JSON body of POST /ai/ask) ───────────────────────────────

// AskResult is the JSON body of POST /ai/ask (contract §5, sprint DoD line 456).
type AskResult struct {
	Intent      string     `json:"intent"`
	Period      PeriodInfo `json:"period"`
	Text        string     `json:"text"`
	Data        any        `json:"data"`
	Suggestions []string   `json:"suggestions"`
	Tier        string     `json:"tier"`
	Confidence  float64    `json:"confidence"`
}

// PeriodInfo is the JSON shape of the resolved period. From/To are omitted
// when the engine could not resolve a date (Query.Period.Unresolved).
type PeriodInfo struct {
	From  *time.Time `json:"from,omitempty"`
	To    *time.Time `json:"to,omitempty"`
	Label string     `json:"label,omitempty"`
}

func buildPeriodInfo(p assistant.Period) PeriodInfo {
	if p.Unresolved {
		return PeriodInfo{}
	}
	from, to := p.From, p.To
	return PeriodInfo{From: &from, To: &to, Label: p.Label}
}

// compareInfo is the "vs the immediately preceding period of the same
// length" figure attached to a numeric intent's data payload when
// Query.Compare is true (AC-8).
type compareInfo struct {
	Previous float64 `json:"previous"`
	DeltaPct float64 `json:"delta_pct"`
}

func newCompareInfo(current, previous float64) *compareInfo {
	return &compareInfo{Previous: previous, DeltaPct: math.Round(deltaPct(current, previous)*100) / 100}
}

type salesData struct {
	Total   float64      `json:"total"`
	Count   int          `json:"count"`
	Compare *compareInfo `json:"compare,omitempty"`
}

type profitData struct {
	Gross            float64      `json:"gross"`
	Expenses         float64      `json:"expenses"`
	Net              float64      `json:"net"`
	UnconfiguredCost int          `json:"unconfigured_cost"`
	Compare          *compareInfo `json:"compare,omitempty"`
}

type marginData struct {
	MarginPct        float64      `json:"margin_pct"`
	TotalRevenue     float64      `json:"total_revenue"`
	UnconfiguredCost int          `json:"unconfigured_cost"`
	Compare          *compareInfo `json:"compare,omitempty"`
}

type topProductsData struct {
	Items []repository.SellerItem `json:"items"`
}

type topMarginData struct {
	Items            []repository.MarginItem `json:"items"`
	UnconfiguredCost int                      `json:"unconfigured_cost"`
}

type lowStockData struct {
	Items []repository.LowStockItem `json:"items"`
}

type debtorsData struct {
	Total float64             `json:"total"`
	Items []repository.Debtor `json:"items"`
}

type expensesData struct {
	Total   float64      `json:"total"`
	Compare *compareInfo `json:"compare,omitempty"`
}

type paymentMixData struct {
	Methods []repository.PaymentMethodTotal `json:"methods"`
}

type cashData struct {
	Estimated float64      `json:"estimated"`
	CashSales float64      `json:"cash_sales"`
	CashOut   float64      `json:"cash_out"`
	Compare   *compareInfo `json:"compare,omitempty"`
}

type creditFlowData struct {
	NewCredit float64      `json:"new_credit"`
	Collected float64      `json:"collected"`
	Compare   *compareInfo `json:"compare,omitempty"`
}

// ─── Service ──────────────────────────────────────────────────────────────

// AssistantService turns a free-form Spanish question (internal/assistant.
// Parse) into a redacted answer with real figures from MetricsRepository/
// ReportRepository/ExpenseRepository, and logs every query (understood or
// not) to AssistantQueryRepository (US-061). It never redefines Query/Intent
// (AC-18): both come from internal/assistant.
type AssistantService struct {
	metrics  repository.MetricsRepository
	reports  repository.ReportRepository
	expenses repository.ExpenseRepository
	queries  repository.AssistantQueryRepository
}

// NewAssistantService constructs an AssistantService.
func NewAssistantService(
	metrics repository.MetricsRepository,
	reports repository.ReportRepository,
	expenses repository.ExpenseRepository,
	queries repository.AssistantQueryRepository,
) *AssistantService {
	return &AssistantService{metrics: metrics, reports: reports, expenses: expenses, queries: queries}
}

// Ask parses query, resolves it against real data and logs the attempt.
// userID is always the caller's own authenticated id (AC-15): it is never
// taken from the query text itself. A failure while logging the bitácora
// (AC-17) is swallowed — it must never surface to the caller nor block the
// computed answer, so it happens after result is already built.
func (s *AssistantService) Ask(ctx context.Context, userID int64, query string) (*AskResult, error) {
	q := assistant.Parse(query, time.Now())

	result, err := s.buildResult(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("assistant ask: %w", err)
	}

	s.logQuery(ctx, userID, query, result.Intent, result.Tier, result.Confidence)
	return result, nil
}

// logQuery inserts the bitácora row. Its error is deliberately ignored
// (AC-17): a broken log must never break the answer already computed.
func (s *AssistantService) logQuery(ctx context.Context, userID int64, query, intent, tier string, confidence float64) {
	_ = s.queries.Create(ctx, userID, query, intent, tier, confidence)
}

// buildResult dispatches on the resolved Query, in priority order: UNKNOWN
// and AMBIGUOUS are terminal engine states independent of any period: then
// an unresolved period (AC-10) short-circuits every real business intent,
// regardless of which one the engine picked; only then do the 11 intents
// compute and format real figures.
func (s *AssistantService) buildResult(ctx context.Context, q assistant.Query) (*AskResult, error) {
	tier := q.Tier
	if tier == "" {
		tier = "none" // the engine never returns Tier="none" itself (contrato §2)
	}

	switch q.Intent {
	case assistant.IntentUnknown:
		r := unknownResult(tier, q.Confidence)
		r.Period = buildPeriodInfo(q.Period)
		return r, nil
	case assistant.IntentAmbiguous:
		r := ambiguousResult(q, tier)
		r.Period = buildPeriodInfo(q.Period)
		return r, nil
	}

	intent := strings.ToLower(string(q.Intent))

	if q.Period.Unresolved {
		r := unresolvedPeriodResult(intent, tier, q.Confidence)
		r.Period = buildPeriodInfo(q.Period)
		return r, nil
	}

	var text string
	var data any
	var err error

	switch q.Intent {
	case assistant.IntentSales:
		text, data, err = s.resolveSales(ctx, q)
	case assistant.IntentProfit:
		text, data, err = s.resolveProfit(ctx, q)
	case assistant.IntentMargin:
		text, data, err = s.resolveMargin(ctx, q)
	case assistant.IntentTopProducts:
		text, data, err = s.resolveTopProducts(ctx, q)
	case assistant.IntentTopMargin:
		text, data, err = s.resolveTopMargin(ctx, q)
	case assistant.IntentLowStock:
		text, data, err = s.resolveLowStock(ctx)
	case assistant.IntentDebtors:
		text, data, err = s.resolveDebtors(ctx)
	case assistant.IntentExpenses:
		text, data, err = s.resolveExpenses(ctx, q)
	case assistant.IntentPaymentMix:
		text, data, err = s.resolvePaymentMix(ctx, q)
	case assistant.IntentCash:
		text, data, err = s.resolveCash(ctx, q)
	case assistant.IntentCreditFlow:
		text, data, err = s.resolveCreditFlow(ctx, q)
	default:
		return nil, fmt.Errorf("unhandled intent %q", q.Intent)
	}
	if err != nil {
		return nil, err
	}

	return &AskResult{
		Intent:      intent,
		Period:      buildPeriodInfo(q.Period),
		Text:        text,
		Data:        data,
		Suggestions: []string{},
		Tier:        tier,
		Confidence:  q.Confidence,
	}, nil
}

// ─── UNKNOWN / AMBIGUOUS / unresolved period ─────────────────────────────────

// unknownSuggestions are the fixed 3 example questions offered on UNKNOWN (AC-6).
var unknownSuggestions = []string{"cuánto vendí hoy", "quién me debe", "qué se está acabando"}

func unknownResult(tier string, confidence float64) *AskResult {
	return &AskResult{
		Intent:      "unknown",
		Text:        "No entendí tu pregunta. Podés intentar con algo como:",
		Suggestions: append([]string{}, unknownSuggestions...),
		Tier:        tier,
		Confidence:  confidence,
	}
}

// intentExample gives a short, human phrase for each real intent, used to
// render AMBIGUOUS candidates (AC-7) as readable suggestions instead of raw
// Intent constants.
var intentExample = map[assistant.Intent]string{
	assistant.IntentSales:       "cuánto vendiste",
	assistant.IntentProfit:      "cuánto ganaste",
	assistant.IntentMargin:      "cuál es tu margen",
	assistant.IntentTopProducts: "qué es lo que más vendés",
	assistant.IntentTopMargin:   "en qué ganás más",
	assistant.IntentLowStock:    "qué se te está acabando",
	assistant.IntentDebtors:     "quién te debe",
	assistant.IntentExpenses:    "cuánto has gastado",
	assistant.IntentPaymentMix:  "cuánto te entró por cada método de pago",
	assistant.IntentCash:        "cuánta plata hay en caja",
	assistant.IntentCreditFlow:  "cuánto fiaste",
}

// intentChip is the tappable chip offered for each AMBIGUOUS candidate. Unlike
// intentExample (2nd person, only used to word the question), every chip is a
// complete phrase the engine resolves back to the same intent (US-062 P-1).
var intentChip = map[assistant.Intent]string{
	assistant.IntentSales:       "cuánto vendí",
	assistant.IntentProfit:      "cuánto gané",
	assistant.IntentMargin:      "cuál es mi margen",
	assistant.IntentTopProducts: "qué es lo que más vendo",
	assistant.IntentTopMargin:   "en qué gano más",
	assistant.IntentLowStock:    "qué se está acabando",
	assistant.IntentDebtors:     "quién me debe",
	assistant.IntentExpenses:    "cuánto he gastado",
	assistant.IntentPaymentMix:  "cuánto me entró por nequi",
	assistant.IntentCash:        "cuánta plata hay en caja",
	assistant.IntentCreditFlow:  "cuánto fié hoy",
}

func ambiguousResult(q assistant.Query, tier string) *AskResult {
	examples := make([]string, 0, len(q.Candidates))
	chips := make([]string, 0, len(q.Candidates))
	for _, c := range q.Candidates {
		if ex, ok := intentExample[c]; ok {
			examples = append(examples, ex)
		}
		if chip, ok := intentChip[c]; ok {
			chips = append(chips, chip)
		}
	}
	return &AskResult{
		Intent:      "ambiguous",
		Text:        "¿Te referís a " + strings.Join(examples, " o a ") + "?",
		Suggestions: chips,
		Tier:        tier,
		Confidence:  q.Confidence,
	}
}

// unresolvedChips holds, per intent (lowercase), the complete phrases
// offered when Period.Unresolved (AC-10, US-062 P-1). Only combinations the
// engine resolves back to the same intent with that period are listed: there
// is no universal period suffix, so some intents offer fewer than 4 chips.
var unresolvedChips = map[string][]string{
	"sales":        {"cuánto vendí hoy", "cuánto vendí ayer", "cuánto vendí esta semana", "cuánto vendí este mes"},
	"profit":       {"cuánto gané hoy", "cuánto gané ayer", "cuánto gané esta semana"},
	"margin":       {"cuál es mi margen hoy", "cuál es mi margen ayer", "cuál es mi margen esta semana", "cuál es mi margen este mes"},
	"top_products": {"cuál es mi producto estrella hoy", "cuál es mi producto estrella ayer", "cuál es mi producto estrella esta semana", "cuál es mi producto estrella este mes"},
	"top_margin":   {"en qué gano más hoy", "en qué gano más ayer"},
	"low_stock":    {"qué se está acabando"},
	"debtors":      {"quién me debe"},
	"expenses":     {"cuánto he gastado hoy", "cuánto he gastado ayer", "cuánto he gastado esta semana", "cuánto he gastado este mes"},
	"payment_mix":  {"cuánto me entró por nequi hoy", "cuánto me entró por nequi ayer", "cuánto me entró por nequi esta semana", "cuánto me entró por nequi este mes"},
	"cash":         {"cuánto efectivo tengo hoy", "cuánto efectivo tengo ayer", "cuánto efectivo tengo esta semana", "cuánto efectivo tengo este mes"},
	"credit_flow":  {"cuánto fié hoy", "cuánto fié ayer", "cuánto fié esta semana"},
}

func unresolvedPeriodResult(intent, tier string, confidence float64) *AskResult {
	return &AskResult{
		Intent:      intent,
		Text:        "¿De qué fecha querés saber? Podés decir hoy, ayer, esta semana o este mes.",
		Suggestions: append([]string{}, unresolvedChips[intent]...),
		Tier:        tier,
		Confidence:  confidence,
	}
}

// ─── Per-intent resolvers ─────────────────────────────────────────────────────

func (s *AssistantService) resolveSales(ctx context.Context, q assistant.Query) (string, any, error) {
	from, to := q.Period.From, q.Period.To
	total, count, err := s.metrics.Sold(ctx, from, to)
	if err != nil {
		return "", nil, err
	}
	if count == 0 {
		return fmt.Sprintf("Todavía no hay ventas registradas %s.", periodPhrase(q.Period.Label)), salesData{}, nil
	}

	if q.Compare {
		pf, pt := previousWindow(from, to)
		prevTotal, prevCount, err := s.metrics.Sold(ctx, pf, pt)
		if err != nil {
			return "", nil, err
		}
		text := fmt.Sprintf("%s vendiste %s en %d %s, frente a %s en %d %s el periodo anterior (%s).",
			q.Period.Label, formatCOP(total), count, pluralize(count, "venta", "ventas"),
			formatCOP(prevTotal), prevCount, pluralize(prevCount, "venta", "ventas"), signedPct(deltaPct(total, prevTotal)))
		return text, salesData{Total: total, Count: count, Compare: newCompareInfo(total, prevTotal)}, nil
	}

	text := fmt.Sprintf("%s vendiste %s en %d %s.", q.Period.Label, formatCOP(total), count, pluralize(count, "venta", "ventas"))
	return text, salesData{Total: total, Count: count}, nil
}

func (s *AssistantService) resolveProfit(ctx context.Context, q assistant.Query) (string, any, error) {
	from, to := q.Period.From, q.Period.To
	_, count, err := s.metrics.Sold(ctx, from, to)
	if err != nil {
		return "", nil, err
	}
	gross, expenses, net, unconfigured, err := s.metrics.Profit(ctx, from, to)
	if err != nil {
		return "", nil, err
	}
	if count == 0 {
		return fmt.Sprintf("Todavía no hay ganancias que mostrar %s: no se registraron ventas.", periodPhrase(q.Period.Label)), profitData{}, nil
	}

	if q.Compare {
		pf, pt := previousWindow(from, to)
		_, _, prevNet, _, err := s.metrics.Profit(ctx, pf, pt)
		if err != nil {
			return "", nil, err
		}
		text := fmt.Sprintf("%s tu ganancia neta fue %s frente a %s el periodo anterior (%s).%s",
			q.Period.Label, formatCOP(net), formatCOP(prevNet), signedPct(deltaPct(net, prevNet)), unconfiguredWarning(unconfigured))
		return text, profitData{Gross: gross, Expenses: expenses, Net: net, UnconfiguredCost: unconfigured, Compare: newCompareInfo(net, prevNet)}, nil
	}

	text := fmt.Sprintf("%s tu ganancia neta fue %s (bruta %s, gastos %s).%s",
		q.Period.Label, formatCOP(net), formatCOP(gross), formatCOP(expenses), unconfiguredWarning(unconfigured))
	return text, profitData{Gross: gross, Expenses: expenses, Net: net, UnconfiguredCost: unconfigured}, nil
}

func (s *AssistantService) resolveMargin(ctx context.Context, q assistant.Query) (string, any, error) {
	from, to := q.Period.From, q.Period.To
	summary, err := s.reports.SalesSummary(ctx, repository.SalesSummaryParams{From: &from, To: &to, GroupBy: "day"})
	if err != nil {
		return "", nil, err
	}
	_, _, _, unconfigured, err := s.metrics.Profit(ctx, from, to)
	if err != nil {
		return "", nil, err
	}
	if summary.TotalCount == 0 {
		return fmt.Sprintf("Todavía no hay ventas %s para calcular el margen.", periodPhrase(q.Period.Label)), marginData{}, nil
	}
	pct := marginPct(summary.GrossMargin, summary.TotalRevenue)

	if q.Compare {
		pf, pt := previousWindow(from, to)
		prevSummary, err := s.reports.SalesSummary(ctx, repository.SalesSummaryParams{From: &pf, To: &pt, GroupBy: "day"})
		if err != nil {
			return "", nil, err
		}
		prevPct := marginPct(prevSummary.GrossMargin, prevSummary.TotalRevenue)
		text := fmt.Sprintf("%s tu margen bruto fue %s%% frente a %s%% el periodo anterior (%s).%s",
			q.Period.Label, formatPct(pct), formatPct(prevPct), signedPct(pct-prevPct), unconfiguredWarning(unconfigured))
		return text, marginData{MarginPct: pct, TotalRevenue: summary.TotalRevenue, UnconfiguredCost: unconfigured, Compare: newCompareInfo(pct, prevPct)}, nil
	}

	text := fmt.Sprintf("%s tu margen bruto fue %s%% sobre %s vendidos.%s",
		q.Period.Label, formatPct(pct), formatCOP(summary.TotalRevenue), unconfiguredWarning(unconfigured))
	return text, marginData{MarginPct: pct, TotalRevenue: summary.TotalRevenue, UnconfiguredCost: unconfigured}, nil
}

// resolveTopProducts ranks by units sold (AC-21, MetricsRepository.TopSellers)
// and adds the volume-vs-margin contrast line (AC-2/AC-3) using TopSeller +
// TopByMargin(1), unchanged from before this US. Compare is not supported for
// this intent (comparing two rankings has no agreed-upon shape, contrato §7).
func (s *AssistantService) resolveTopProducts(ctx context.Context, q assistant.Query) (string, any, error) {
	from, to := q.Period.From, q.Period.To
	sellers, err := s.metrics.TopSellers(ctx, from, to, assistantFetchLimit)
	if err != nil {
		return "", nil, err
	}
	if len(sellers) == 0 {
		return fmt.Sprintf("Todavía no vendiste nada %s.", periodPhrase(q.Period.Label)), topProductsData{Items: []repository.SellerItem{}}, nil
	}

	shown, more := truncate(sellers, assistantDisplayLimit)
	var b strings.Builder
	fmt.Fprintf(&b, "%s lo que más vendiste fue:\n", q.Period.Label)
	for i, item := range shown {
		fmt.Fprintf(&b, "%d. %s — %s unidades (%s)\n", i+1, item.Name, formatQty(item.Quantity), formatCOP(item.Revenue))
	}
	if more > 0 {
		fmt.Fprintf(&b, "y %d más\n", more)
	}

	topSellerName, topSellerQty, err := s.metrics.TopSeller(ctx, from, to)
	if err != nil {
		return "", nil, err
	}
	topMargin, err := s.metrics.TopByMargin(ctx, from, to, 1)
	if err != nil {
		return "", nil, err
	}
	if len(topMargin) > 0 && topSellerName != "" && topMargin[0].Name != topSellerName {
		fmt.Fprintf(&b, "Lo que más vendés es %s (%s unidades), pero lo que más plata te deja es %s (%s%% de margen).",
			topSellerName, formatQty(topSellerQty), topMargin[0].Name, formatPct(topMargin[0].MarginPct))
	}

	return strings.TrimRight(b.String(), "\n"), topProductsData{Items: shown}, nil
}

func (s *AssistantService) resolveTopMargin(ctx context.Context, q assistant.Query) (string, any, error) {
	from, to := q.Period.From, q.Period.To
	items, err := s.metrics.TopByMargin(ctx, from, to, assistantFetchLimit)
	if err != nil {
		return "", nil, err
	}
	_, _, _, unconfigured, err := s.metrics.Profit(ctx, from, to)
	if err != nil {
		return "", nil, err
	}
	if len(items) == 0 {
		return fmt.Sprintf("Todavía no hay ventas con margen que mostrar %s.", periodPhrase(q.Period.Label)), topMarginData{Items: []repository.MarginItem{}}, nil
	}

	shown, more := truncate(items, assistantDisplayLimit)
	var b strings.Builder
	fmt.Fprintf(&b, "%s lo que más plata te deja es:\n", q.Period.Label)
	for i, it := range shown {
		fmt.Fprintf(&b, "%d. %s — %s%% de margen (%s unidades)\n", i+1, it.Name, formatPct(it.MarginPct), formatQty(it.UnitsSold))
	}
	if more > 0 {
		fmt.Fprintf(&b, "y %d más\n", more)
	}
	text := strings.TrimRight(b.String(), "\n") + unconfiguredWarning(unconfigured)
	return text, topMarginData{Items: shown, UnconfiguredCost: unconfigured}, nil
}

// resolveLowStock and resolveDebtors ignore Period on purpose: both reflect
// the CURRENT state of inventory/portfolio, not a date-ranged figure — the
// underlying MetricsRepository methods do not take a [from, to) range
// either. AC-9 ("nombrar el periodo") therefore does not apply to them.
func (s *AssistantService) resolveLowStock(ctx context.Context) (string, any, error) {
	items, err := s.metrics.LowStock(ctx, assistantFetchLimit)
	if err != nil {
		return "", nil, err
	}
	if len(items) == 0 {
		return "Por ahora ningún producto está por debajo de su stock mínimo.", lowStockData{Items: []repository.LowStockItem{}}, nil
	}

	shown, more := truncate(items, assistantDisplayLimit)
	var b strings.Builder
	b.WriteString("Se te está acabando:\n")
	for i, it := range shown {
		fmt.Fprintf(&b, "%d. %s (quedan %s, mínimo %s)\n", i+1, it.Name, formatQty(it.Stock), formatQty(it.MinStock))
	}
	if more > 0 {
		fmt.Fprintf(&b, "y %d más\n", more)
	}
	return strings.TrimRight(b.String(), "\n"), lowStockData{Items: shown}, nil
}

func (s *AssistantService) resolveDebtors(ctx context.Context) (string, any, error) {
	debtors, err := s.metrics.Debtors(ctx, 0) // 0 == no limit (metricsRepo.Debtors convention)
	if err != nil {
		return "", nil, err
	}
	if len(debtors) == 0 {
		return "Nadie te debe plata ahora mismo. Cartera en cero.", debtorsData{Items: []repository.Debtor{}}, nil
	}

	var total float64
	for _, d := range debtors {
		total += d.Balance
	}
	shown, more := truncate(debtors, assistantDisplayLimit)
	var b strings.Builder
	fmt.Fprintf(&b, "Te deben en total %s:\n", formatCOP(total))
	for i, d := range shown {
		fmt.Fprintf(&b, "%d. %s — %s\n", i+1, d.Name, formatCOP(d.Balance))
	}
	if more > 0 {
		fmt.Fprintf(&b, "y %d más\n", more)
	}
	return strings.TrimRight(b.String(), "\n"), debtorsData{Total: total, Items: shown}, nil
}

func (s *AssistantService) resolveExpenses(ctx context.Context, q assistant.Query) (string, any, error) {
	from, to := q.Period.From, q.Period.To
	sum, err := s.expenses.SumRange(ctx, from, to)
	if err != nil {
		return "", nil, err
	}
	if sum == 0 {
		return fmt.Sprintf("No registraste gastos %s.", periodPhrase(q.Period.Label)), expensesData{}, nil
	}

	if q.Compare {
		pf, pt := previousWindow(from, to)
		prevSum, err := s.expenses.SumRange(ctx, pf, pt)
		if err != nil {
			return "", nil, err
		}
		text := fmt.Sprintf("%s gastaste %s frente a %s el periodo anterior (%s).",
			q.Period.Label, formatCOP(sum), formatCOP(prevSum), signedPct(deltaPct(sum, prevSum)))
		return text, expensesData{Total: sum, Compare: newCompareInfo(sum, prevSum)}, nil
	}

	text := fmt.Sprintf("%s gastaste %s.", q.Period.Label, formatCOP(sum))
	return text, expensesData{Total: sum}, nil
}

// nequiClarification is the fixed disclaimer AC-22 requires on PAYMENT_MIX:
// "transferencia" is a bucket, not a real distinction between Nequi/
// Daviplata/other electronic wallets (glosario.md, D-1 of this US's contract).
const nequiClarification = "\nOjo: \"transferencia\" agrupa Nequi, Daviplata y otros medios electrónicos; hoy no se distinguen por separado."

func (s *AssistantService) resolvePaymentMix(ctx context.Context, q assistant.Query) (string, any, error) {
	from, to := q.Period.From, q.Period.To
	summary, err := s.reports.SalesSummary(ctx, repository.SalesSummaryParams{From: &from, To: &to, GroupBy: "day"})
	if err != nil {
		return "", nil, err
	}
	if len(summary.ByPaymentMethod) == 0 {
		return fmt.Sprintf("Todavía no hay pagos registrados %s.", periodPhrase(q.Period.Label)), paymentMixData{Methods: []repository.PaymentMethodTotal{}}, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s te entró:\n", q.Period.Label)
	for _, pm := range summary.ByPaymentMethod {
		fmt.Fprintf(&b, "- %s: %s\n", paymentMethodLabel(pm.Method), formatCOP(pm.Total))
	}
	text := strings.TrimRight(b.String(), "\n") + nequiClarification
	return text, paymentMixData{Methods: summary.ByPaymentMethod}, nil
}

func (s *AssistantService) resolveCash(ctx context.Context, q assistant.Query) (string, any, error) {
	from, to := q.Period.From, q.Period.To
	cashSales, cashOut, err := s.metrics.CashFlow(ctx, from, to)
	if err != nil {
		return "", nil, err
	}
	if cashSales == 0 && cashOut == 0 {
		return fmt.Sprintf("No hay movimientos de caja %s.", periodPhrase(q.Period.Label)), cashData{}, nil
	}
	estimated := cashSales - cashOut

	if q.Compare {
		pf, pt := previousWindow(from, to)
		prevCashSales, prevCashOut, err := s.metrics.CashFlow(ctx, pf, pt)
		if err != nil {
			return "", nil, err
		}
		prevEstimated := prevCashSales - prevCashOut
		text := fmt.Sprintf("%s tenés aproximadamente %s en caja frente a %s el periodo anterior (%s).",
			q.Period.Label, formatCOP(estimated), formatCOP(prevEstimated), signedPct(deltaPct(estimated, prevEstimated)))
		return text, cashData{Estimated: estimated, CashSales: cashSales, CashOut: cashOut, Compare: newCompareInfo(estimated, prevEstimated)}, nil
	}

	text := fmt.Sprintf("%s tenés aproximadamente %s en caja (ventas en efectivo %s, salidas %s).",
		q.Period.Label, formatCOP(estimated), formatCOP(cashSales), formatCOP(cashOut))
	return text, cashData{Estimated: estimated, CashSales: cashSales, CashOut: cashOut}, nil
}

func (s *AssistantService) resolveCreditFlow(ctx context.Context, q assistant.Query) (string, any, error) {
	from, to := q.Period.From, q.Period.To
	newCredit, collected, err := s.metrics.CreditFlow(ctx, from, to)
	if err != nil {
		return "", nil, err
	}
	if newCredit == 0 && collected == 0 {
		return fmt.Sprintf("No fiaste ni te abonaron nada %s.", periodPhrase(q.Period.Label)), creditFlowData{}, nil
	}

	if q.Compare {
		pf, pt := previousWindow(from, to)
		prevNewCredit, _, err := s.metrics.CreditFlow(ctx, pf, pt)
		if err != nil {
			return "", nil, err
		}
		text := fmt.Sprintf("%s fiaste %s frente a %s el periodo anterior (%s).",
			q.Period.Label, formatCOP(newCredit), formatCOP(prevNewCredit), signedPct(deltaPct(newCredit, prevNewCredit)))
		return text, creditFlowData{NewCredit: newCredit, Collected: collected, Compare: newCompareInfo(newCredit, prevNewCredit)}, nil
	}

	text := fmt.Sprintf("%s fiaste %s y te abonaron %s.", q.Period.Label, formatCOP(newCredit), formatCOP(collected))
	return text, creditFlowData{NewCredit: newCredit, Collected: collected}, nil
}

// ─── Formatting helpers ───────────────────────────────────────────────────────

// formatCOP renders n as Colombian pesos: no decimals, "." as thousands
// separator, "$" prefix, rounded with math.Round (AC-11). Mirrors
// frontend/lib/format.ts formatCOP and internal/mailer's private copy — kept
// separate on purpose (contrato §2: no reusable helper exists yet across
// packages; unifying them is not in this US's scope).
func formatCOP(n float64) string {
	rounded := int64(math.Round(n))
	neg := rounded < 0
	if neg {
		rounded = -rounded
	}
	digits := strconv.FormatInt(rounded, 10)
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

// formatQty renders a quantity without a trailing ".00" for whole units
// (most products are sold by unit), but keeps 2 decimals for fractional
// ones (kg/lb/liter).
func formatQty(v float64) string {
	if v == math.Trunc(v) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// formatPct renders a percentage rounded to 1 decimal, without a trailing
// ".0" when it lands on a whole number.
func formatPct(v float64) string {
	return strconv.FormatFloat(math.Round(v*10)/10, 'f', -1, 64)
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

// periodPhrase turns a Period.Label into the lowercase phrase used inside a
// sentence ("Hoy" -> "hoy", but "Mes pasado" -> "el mes pasado", matching
// natural Spanish).
func periodPhrase(label string) string {
	switch label {
	case "Hoy":
		return "hoy"
	case "Ayer":
		return "ayer"
	case "Esta semana":
		return "esta semana"
	case "Semana pasada":
		return "la semana pasada"
	case "Este mes":
		return "este mes"
	case "Mes pasado":
		return "el mes pasado"
	case "Este año":
		return "este año"
	default:
		return "en este periodo"
	}
}

// unconfiguredWarning is AC-4's advisory, appended verbatim to PROFIT,
// MARGIN and TOP_MARGIN answers when unconfiguredCost > 0 (sprint DoD line
// 387-390). Empty when there is nothing to warn about.
func unconfiguredWarning(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("\nOjo: %d productos no tienen costo configurado, así que la ganancia real puede ser mayor.", n)
}

var paymentMethodLabels = map[string]string{
	"cash":     "Efectivo",
	"transfer": "Transferencia",
	"card":     "Tarjeta",
	"credit":   "Fiado",
}

func paymentMethodLabel(method string) string {
	if label, ok := paymentMethodLabels[method]; ok {
		return label
	}
	return method
}

// marginPct is gross margin as a percentage of revenue; 0 when there is no
// revenue to divide by (avoids NaN/Inf).
func marginPct(grossMargin, totalRevenue float64) float64 {
	if totalRevenue == 0 {
		return 0
	}
	return grossMargin / totalRevenue * 100
}

// previousWindow returns the immediately preceding period of the same
// length as [from, to) — e.g. for "hoy" ([T, T+1d)) it returns "ayer"
// ([T-1d, T)) — used by Compare (AC-8).
func previousWindow(from, to time.Time) (time.Time, time.Time) {
	length := to.Sub(from)
	return from.Add(-length), from
}

// deltaPct is the percentage change from previous to current. previous == 0
// is treated as "all new" (+100%) rather than dividing by zero, except when
// both are 0 (no change, 0%).
func deltaPct(current, previous float64) float64 {
	if previous == 0 {
		if current == 0 {
			return 0
		}
		return 100
	}
	return (current - previous) / previous * 100
}

// signedPct formats a percentage with its sign ALWAYS visible (AC-8,
// decision GATE 1): "+15%" on the rise, "-20%" on a drop, "+0%" on no
// change — never a bare number.
func signedPct(v float64) string {
	r := int64(math.Round(v))
	sign := "+"
	if r < 0 {
		sign = "-"
		r = -r
	}
	return fmt.Sprintf("%s%d%%", sign, r)
}

// truncate splits items into the first `limit` elements to display and how
// many more remain, for the "y N más" summary (sprint DoD line 392-395).
func truncate[T any](items []T, limit int) ([]T, int) {
	if len(items) <= limit {
		return items, 0
	}
	return items[:limit], len(items) - limit
}
