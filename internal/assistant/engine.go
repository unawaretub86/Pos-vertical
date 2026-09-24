package assistant

import (
	"regexp"
	"strings"
	"time"
)

// Intent is one of the 11 questions a store owner can ask the local
// assistant, plus the two terminal states UNKNOWN/AMBIGUOUS.
type Intent string

const (
	IntentSales       Intent = "SALES"
	IntentProfit      Intent = "PROFIT"
	IntentMargin      Intent = "MARGIN"
	IntentTopProducts Intent = "TOP_PRODUCTS"
	IntentTopMargin   Intent = "TOP_MARGIN"
	IntentLowStock    Intent = "LOW_STOCK"
	IntentDebtors     Intent = "DEBTORS"
	IntentExpenses    Intent = "EXPENSES"
	IntentPaymentMix  Intent = "PAYMENT_MIX"
	IntentCash        Intent = "CASH"
	IntentCreditFlow  Intent = "CREDIT_FLOW"
	IntentUnknown     Intent = "UNKNOWN"
	IntentAmbiguous   Intent = "AMBIGUOUS"
)

// Query is the structured result of Parse: what the store owner wants to
// know (Intent), over what date range (Period), whether they asked for a
// comparison (Compare), and how confident the engine is (Confidence/Tier).
// Entity exists per the contract (§7: "nace presente pero sin usar") but is
// never populated by this US — reserved for a future single-product query.
type Query struct {
	Intent     Intent
	Period     Period
	Compare    bool
	Entity     string
	Confidence float64
	Tier       string
	Candidates []Intent
}

const (
	fuzzyThreshold   = 0.60
	ambiguousMargin  = 0.10
	maxFuzzyTokenLen = 64 // guards against pathological input (AC-20)
)

// exactSynonyms are known typo pairs whose real Levenshtein distance
// exceeds the general <=1 fallback (decision GATE 1, contrato §2).
// "kien"->"quien" is cited literally by the contract (AC-5, T-5, T-27):
// levenshtein("kien","quien")==2, outside the general threshold. Checked
// BEFORE the levenshtein<=1 fallback in coincide(). (period.go keeps its
// own periodSynonyms map for "aller"->"ayer" — a separate vocabulary, per
// decision GATE 1 §2.)
var exactSynonyms = map[string]string{
	"kien": "quien",
}

// compareTriggers (AC-17) are matched as whole words/phrases against the
// pre-stopword-stripped form of the query (stripAccentsLowerPunct), because
// some of them rely on "que", which normalize() removes as a stopword.
var compareTriggers = []string{
	"mas que", "menos que", "comparado", "vs", "que ayer", "que el mes pasado",
}

// intentDef is one row of the intent registry: an explicit, ordered
// definition (AC-4/AC-23 forbid resolving ties via map iteration). phrases
// feeds the T1 anchor pattern (comprehensive: every canonical + variant the
// bank must resolve via regex); fuzzySeeds is a smaller, curated set used
// for T2 scoring only. They are kept separate on purpose: a long literal
// T1-only phrase (e.g. "quien debe plata") can make a typo'd query score a
// coincidental 1.0 in T2 if reused as a fuzzy seed, which would violate
// AC-5's "Confidence < 1.0 en Tier=fuzzy" invariant.
type intentDef struct {
	intent   Intent
	phrases  []string
	pattern  *regexp.Regexp
	seedToks [][]string
}

// intentRegistry is the ordered, explicit list of intents (AC-4, AC-23),
// populated by buildIntentRegistry below.
var intentRegistry = buildIntentRegistry()

// ambiguousSeed is added to both SALES and PROFIT on purpose: it is the
// literal example the contract uses for AC-4/T-4 ("cuanto gane vendiendo"),
// a phrase that genuinely matches both banks. Whichever of the two is
// registered first in buildIntentRegistry wins, deterministically (AC-4).
const ambiguousSeed = "cuanto gane vendiendo"

// buildIntentRegistry is the single ordered source of truth for T1 (regex)
// and T2 (fuzzy) matching (AC-4, AC-23): every phrase below feeds both an
// anchored regex alternative (T1) and a tokenized fuzzy seed (T2), so the
// two tiers can never drift apart. Phrases are literal examples from the
// glosario/sprint bank (spec/domain/glosario.md:6) plus their obvious
// variants; contract §5 leaves this bank to the implementer's judgement
// (GATE 1, pregunta 3).
func buildIntentRegistry() []intentDef {
	return []intentDef{
		newIntentDef(IntentSales, []string{
			"cuánto vendí", "cuánto vendí hoy", "cuánto facturé",
			"cómo van las ventas", "cuánto he vendido", "cuánto vendí este mes",
			// The 4 below cover the remaining Period.Label values (US-061
			// AC-9/T-9: hoy, ayer, este mes ya resolvían via T1/T2; estos
			// completan esta semana/semana pasada/mes pasado/este año, que
			// el bench original de US-060 no cubría literalmente).
			"cuánto vendí esta semana", "cuánto vendí la semana pasada",
			"cuánto vendí el mes pasado", "cuánto vendí este año",
			ambiguousSeed,
		}, []string{"cuánto vendí"}),
		newIntentDef(IntentProfit, []string{
			"cuánto gané", "cuánta ganancia tuve", "cuánto me quedó limpio",
			"cuál fue mi ganancia", "cuánto gané hoy", "cuánto gané esta semana",
			ambiguousSeed,
		}, []string{"cuánto gané"}),
		newIntentDef(IntentMargin, []string{
			"cuál es mi margen", "qué porcentaje gano", "cuál es mi margen de ganancia",
			"qué margen tengo", "cuánto es mi margen bruto", "cuál es mi margen hoy",
		}, []string{"cuál es mi margen"}),
		newIntentDef(IntentTopProducts, []string{
			"qué es lo que más vendo", "cuál es mi producto estrella", "qué producto vendo más",
			"cuáles son mis productos más vendidos", "qué es lo más vendido",
		}, []string{"cuál es mi producto estrella"}),
		newIntentDef(IntentTopMargin, []string{
			"qué me deja más plata", "en qué gano más", "qué producto me deja más ganancia",
			"en qué producto gano más plata", "cuál es el producto que más me deja",
		}, []string{"en qué gano más"}),
		newIntentDef(IntentLowStock, []string{
			"qué se me está acabando", "qué tengo que pedir", "qué productos se están agotando",
			"qué me está haciendo falta en el inventario", "qué se está acabando",
		}, []string{"qué se está acabando"}),
		newIntentDef(IntentDebtors, []string{
			"quién me debe", "cómo va la cartera", "a quién le fié",
			"quiénes me deben plata", "cómo está mi cartera", "quién me debe plata",
		}, []string{"quién me debe"}),
		newIntentDef(IntentExpenses, []string{
			"cuánto he gastado", "cuánto llevo en gastos", "cuánto gasté",
			"cuáles son mis gastos", "cuánto he gastado este mes",
		}, []string{"cuánto he gastado"}),
		newIntentDef(IntentPaymentMix, []string{
			"cuánto me entró por nequi", "cuánto en efectivo me entró", "cuánto me pagaron por transferencia",
			"cuánto recibí por nequi", "cuánto entró por nequi hoy", "cuánto me entró por transferencia",
		}, []string{"cuánto me entró por nequi"}),
		newIntentDef(IntentCash, []string{
			"cuánta plata hay en caja", "cuánto efectivo tengo", "cuánto hay en caja",
			"cuánta plata tengo en caja", "cuánto efectivo hay disponible",
		}, []string{"cuánto efectivo tengo"}),
		newIntentDef(IntentCreditFlow, []string{
			"cuánto fié hoy", "cuánto me abonaron", "cuánto fié esta semana",
			"cuánto me han abonado", "cuánto llevo fiado hoy", "cuánto me abonaron hoy",
		}, []string{"cuánto fié hoy"}),
	}
}

func newIntentDef(intent Intent, phrases []string, fuzzySeeds []string) intentDef {
	normed := make([]string, len(phrases))
	for i, p := range phrases {
		normed[i] = regexp.QuoteMeta(normalize(p))
	}
	seedToks := make([][]string, len(fuzzySeeds))
	for i, p := range fuzzySeeds {
		seedToks[i] = strings.Fields(normalize(p))
	}
	pattern := regexp.MustCompile("^(" + strings.Join(normed, "|") + ")$")
	return intentDef{intent: intent, phrases: phrases, pattern: pattern, seedToks: seedToks}
}

// levenshtein computes the classic edit distance (insert/delete/substitute,
// cost 1 each) between two strings, rune-aware so accented/ñ characters
// count as a single edit unit.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			best := del
			if ins < best {
				best = ins
			}
			if sub < best {
				best = sub
			}
			curr[j] = best
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

// coincide(t,s) implements the T2 matching rule decided at GATE 1 (§2 del
// contrato): igualdad exacta -> sinonimos exactos -> levenshtein<=1 con
// len(t)>=4 como respaldo. t is the query token, s is the seed token.
func coincide(t, s string) bool {
	if t == s {
		return true
	}
	if syn, ok := exactSynonyms[t]; ok && syn == s {
		return true
	}
	if len(t) < 4 || len(t) > maxFuzzyTokenLen {
		return false
	}
	return levenshtein(t, s) <= 1
}

// fuzzyScore is (tokens que coinciden) / MAX(tokens semilla, tokens
// consulta), AC-5.
func fuzzyScore(seed []string, query []string) float64 {
	matched := 0
	for _, s := range seed {
		for _, t := range query {
			if coincide(t, s) {
				matched++
				break
			}
		}
	}
	denom := len(seed)
	if len(query) > denom {
		denom = len(query)
	}
	if denom == 0 {
		return 0
	}
	return float64(matched) / float64(denom)
}

// bestScore returns the best fuzzy score of any seed phrase belonging to
// this intent against the query tokens.
func (d intentDef) bestScore(query []string) float64 {
	best := 0.0
	for _, seed := range d.seedToks {
		if sc := fuzzyScore(seed, query); sc > best {
			best = sc
		}
	}
	return best
}

// splitAtCompareTrigger covers AC-17. It looks for a comparison trigger
// (AC-17: "mas que", "menos que", "comparado", "vs", "que ayer", "que el
// mes pasado") in a lightly normalized form of the query (accents/case/
// punctuation stripped, but stopwords like "que" kept — some triggers
// depend on it), and returns the portion of the query BEFORE the earliest
// trigger. Intent and Period are resolved from that portion only: the
// comparison target ("...comparado con ayer") is what we compare AGAINST,
// not the period being asked about (AC-17: "conservando el Intent y el
// Period detectados de forma independiente").
func splitAtCompareTrigger(raw string) (before string, compare bool) {
	processed := stripAccentsLowerPunct(raw)
	padded := " " + processed + " "
	cut := -1
	for _, trig := range compareTriggers {
		needle := " " + trig + " "
		idx := strings.Index(padded, needle)
		if idx == -1 {
			continue
		}
		compare = true
		pos := idx // position within padded; padded[0] is the extra leading space, so idx already equals the cut point within processed
		if cut == -1 || pos < cut {
			cut = pos
		}
	}
	if cut < 0 {
		return processed, compare
	}
	return strings.TrimSpace(padded[:cut]), compare
}

// Parse translates a free-form Spanish question into a Query. It is the
// ONLY exported function of the engine (AC-22); now is injected so tests
// can freeze "the present" for deterministic relative periods (AC-8..AC-16).
func Parse(q string, now time.Time) Query {
	prefix, compare := splitAtCompareTrigger(q)
	norm := normalize(prefix)
	tokens := strings.Fields(norm)
	period := extractPeriod(tokens, now)

	if norm != "" {
		for _, def := range intentRegistry {
			if def.pattern.MatchString(norm) {
				return Query{Intent: def.intent, Period: period, Compare: compare, Confidence: 1.0, Tier: "regex"}
			}
		}
	}

	if len(tokens) == 0 {
		return Query{Intent: IntentUnknown, Period: period, Compare: compare}
	}

	bestIntent := IntentUnknown
	bestScoreVal := 0.0
	secondScoreVal := 0.0
	for _, def := range intentRegistry {
		sc := def.bestScore(tokens)
		if sc > bestScoreVal {
			secondScoreVal = bestScoreVal
			bestScoreVal = sc
			bestIntent = def.intent
		} else if sc > secondScoreVal {
			secondScoreVal = sc
		}
	}

	if bestScoreVal < fuzzyThreshold {
		return Query{Intent: IntentUnknown, Period: period, Compare: compare, Confidence: bestScoreVal}
	}

	if secondScoreVal >= fuzzyThreshold && (bestScoreVal-secondScoreVal) < ambiguousMargin {
		candidates := ambiguousCandidates(tokens, bestScoreVal, secondScoreVal)
		return Query{Intent: IntentAmbiguous, Period: period, Compare: compare, Confidence: bestScoreVal, Tier: "fuzzy", Candidates: candidates}
	}

	return Query{Intent: bestIntent, Period: period, Compare: compare, Confidence: bestScoreVal, Tier: "fuzzy"}
}

// ambiguousCandidates re-walks the registry (in its fixed order, AC-23) to
// collect the intents whose score is within ambiguousMargin of the top
// score, per AC-7 ("los dos candidatos"). Open question in contract.md §8
// (non-blocking): only top-2 ties are contemplated; if 3+ intents ever tie
// this returns them all, which the current tests never exercise.
func ambiguousCandidates(tokens []string, best, second float64) []Intent {
	var out []Intent
	for _, def := range intentRegistry {
		sc := def.bestScore(tokens)
		if sc >= fuzzyThreshold && (best-sc) < ambiguousMargin+1e-9 {
			out = append(out, def.intent)
		}
	}
	return out
}
