package assistant

// Tests del motor de intenciones (T1 regex + T2 fuzzy) — contrato US-060
// §3/§4. fixedNow y midnight() estan definidos en period_test.go (mismo
// paquete).
//
// DECISIONES DE DISEÑO DEL TEST-AUTHOR (documentadas, no silenciadas):
//
// 1. El banco de "≥60 frases" de T-3/AC-24 se construye SOLO con frases
//    pensadas para resolver via T1 (regex), usando literalmente las frases
//    de ejemplo de la columna "Responde a" de la tabla de 11 intents del
//    sprint (lineas 265-275) mas variantes de fraseo natural. NO se
//    incluyen alli variantes con errores de dedo, porque su resolucion
//    dependeria del banco semilla fuzzy interno que el implementer aun no
//    ha escrito (GATE1, pregunta 3: los sinonimos coloquiales por intent
//    son libres del implementer) y el contrato no fija esas frases semilla
//    palabra por palabra salvo los 4 casos de AC-5. Cubrir "con errores de
//    dedo" (AC-24) se resuelve en conjunto con T-5, T-13, T-14, T-26, T-27
//    y T-28, que SI usan frases con errores de dedo literalmente citadas
//    o exigidas por el contrato.
//
// 2. T-24 (no-solapamiento entre bancos, AC-25) reutiliza la misma tabla
//    de T-3 en lugar de acceder a un "banco semilla" interno sin nombre
//    fijado por el contrato: cada fila de T-3 ya declara su intent
//    esperado, asi que "ninguna frase de un banco resuelve a otro intent"
//    se verifica corriendo Parse sobre esa misma tabla. Esto evita
//    inventar un simbolo interno (p. ej. un `seedBanks` exportado o no)
//    que el contrato no nombra.
//
// 3. T-7 (AMBIGUOUS) requiere una frase que empate dos intents dentro de
//    0.10 en el motor fuzzy. Esto depende del banco semilla interno del
//    implementer, que el contrato no fija para este caso (a diferencia de
//    los 4 casos obligatorios de AC-5). Se construyo la frase
//    "cuanto bendi y gaine" asumiendo que el banco semilla de SALES
//    incluye "cuanto vendi" y el de PROFIT incluye "cuanto gane" (las
//    frases canonicas literales del sprint, lineas 265-266). Si el
//    implementer elige otro fraseo semilla, este test puede necesitar
//    ajustar la frase de entrada — el invariante que importa (AMBIGUOUS
//    con 2 candidatos) esta documentado igual en el nombre del test.
//
// 4. T-4 no fuerza que "cuanto gane vendiendo" resuelva a SALES o a
//    PROFIT especificamente: el contrato no fija el orden del registro de
//    intents (eso es libertad del implementer, siempre que sea un slice
//    declarado y no un map). El test solo exige que el resultado sea uno
//    de los dos y que sea ESTABLE en 100+ corridas, que es lo que AC-4/
//    AC-23 realmente piden probar.

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

type engineCase struct {
	input  string
	intent Intent
}

// engineT3Cases: banco de ≥60 frases para las 11 intenciones (T-3/AC-24).
// Todas pensadas para resolver via T1 (regex), Tier="regex", Confidence=1.0.
var engineT3Cases = []engineCase{
	// SALES
	{"cuánto vendí", IntentSales},
	{"cuánto vendí hoy", IntentSales},
	{"cuánto facturé", IntentSales},
	{"cómo van las ventas", IntentSales},
	{"cuánto he vendido", IntentSales},
	{"¿CUÁNTO VENDÍ ESTE MES?", IntentSales},

	// PROFIT
	{"cuánto gané", IntentProfit},
	{"cuánta ganancia tuve", IntentProfit},
	{"cuánto me quedó limpio", IntentProfit},
	{"cuál fue mi ganancia", IntentProfit},
	{"cuánto gané hoy", IntentProfit},
	{"¿CUÁNTO GANÉ ESTA SEMANA?", IntentProfit},

	// MARGIN
	{"cuál es mi margen", IntentMargin},
	{"qué porcentaje gano", IntentMargin},
	{"cuál es mi margen de ganancia", IntentMargin},
	{"qué margen tengo", IntentMargin},
	{"cuánto es mi margen bruto", IntentMargin},
	{"¿CUÁL ES MI MARGEN HOY?", IntentMargin},

	// TOP_PRODUCTS
	{"qué es lo que más vendo", IntentTopProducts},
	{"cuál es mi producto estrella", IntentTopProducts},
	{"qué producto vendo más", IntentTopProducts},
	{"cuáles son mis productos más vendidos", IntentTopProducts},
	{"qué es lo más vendido", IntentTopProducts},
	{"¿CUÁL ES MI PRODUCTO ESTRELLA?", IntentTopProducts},

	// TOP_MARGIN
	{"qué me deja más plata", IntentTopMargin},
	{"en qué gano más", IntentTopMargin},
	{"qué producto me deja más ganancia", IntentTopMargin},
	{"en qué producto gano más plata", IntentTopMargin},
	{"cuál es el producto que más me deja", IntentTopMargin},
	{"¿EN QUÉ GANO MÁS?", IntentTopMargin},

	// LOW_STOCK
	{"qué se me está acabando", IntentLowStock},
	{"qué tengo que pedir", IntentLowStock},
	{"qué productos se están agotando", IntentLowStock},
	{"qué me está haciendo falta en el inventario", IntentLowStock},
	{"qué se está acabando", IntentLowStock},
	{"¿QUÉ TENGO QUE PEDIR?", IntentLowStock},

	// DEBTORS
	{"quién me debe", IntentDebtors},
	{"cómo va la cartera", IntentDebtors},
	{"a quién le fié", IntentDebtors},
	{"quiénes me deben plata", IntentDebtors},
	{"cómo está mi cartera", IntentDebtors},
	{"¿QUIÉN ME DEBE PLATA?", IntentDebtors},

	// EXPENSES
	{"cuánto he gastado", IntentExpenses},
	{"cuánto llevo en gastos", IntentExpenses},
	{"cuánto gasté", IntentExpenses},
	{"cuáles son mis gastos", IntentExpenses},
	{"cuánto he gastado este mes", IntentExpenses},
	{"¿CUÁNTO LLEVO EN GASTOS?", IntentExpenses},

	// PAYMENT_MIX
	{"cuánto me entró por nequi", IntentPaymentMix},
	{"cuánto en efectivo me entró", IntentPaymentMix},
	{"cuánto me pagaron por transferencia", IntentPaymentMix},
	{"cuánto recibí por nequi", IntentPaymentMix},
	{"cuánto entró por nequi hoy", IntentPaymentMix},
	{"¿CUÁNTO ME ENTRÓ POR TRANSFERENCIA?", IntentPaymentMix},

	// CASH
	{"cuánta plata hay en caja", IntentCash},
	{"cuánto efectivo tengo", IntentCash},
	{"cuánto hay en caja", IntentCash},
	{"cuánta plata tengo en caja", IntentCash},
	{"cuánto efectivo hay disponible", IntentCash},
	{"¿CUÁNTA PLATA HAY EN CAJA?", IntentCash},

	// CREDIT_FLOW
	{"cuánto fié hoy", IntentCreditFlow},
	{"cuánto me abonaron", IntentCreditFlow},
	{"cuánto fié esta semana", IntentCreditFlow},
	{"cuánto me han abonado", IntentCreditFlow},
	{"cuánto llevo fiado hoy", IntentCreditFlow},
	{"¿CUÁNTO ME ABONARON HOY?", IntentCreditFlow},
}

// TestEngine_T3_ElevenIntentsMinimum60Phrases cubre AC-3 y AC-24: ≥60
// frases, 11 intenciones, cada fila via T1 con Confidence=1.0 y
// Tier="regex".
func TestEngine_T3_ElevenIntentsMinimum60Phrases(t *testing.T) {
	if len(engineT3Cases) < 60 {
		t.Fatalf("el banco de frases tiene %d filas, se requieren >= 60 (AC-24)", len(engineT3Cases))
	}
	intentsSeen := map[Intent]bool{}
	for _, tc := range engineT3Cases {
		intentsSeen[tc.intent] = true
		t.Run(tc.input, func(t *testing.T) {
			got := Parse(tc.input, fixedNow)
			if got.Intent != tc.intent {
				t.Errorf("Parse(%q).Intent = %s, want %s", tc.input, got.Intent, tc.intent)
			}
			if got.Tier != "regex" {
				t.Errorf("Parse(%q).Tier = %q, want \"regex\"", tc.input, got.Tier)
			}
			if got.Confidence != 1.0 {
				t.Errorf("Parse(%q).Confidence = %v, want 1.0", tc.input, got.Confidence)
			}
		})
	}
	if len(intentsSeen) != 11 {
		t.Errorf("el banco cubre %d intents distintos, se requieren las 11 intenciones", len(intentsSeen))
	}
}

// TestEngine_T4_AmbiguousRegexOrderIsStableAndDeclared cubre AC-4: el
// registro de intents debe tener orden fijo (slice, no map), asi que una
// frase que podria matchear dos patrones (SALES/PROFIT) siempre resuelve
// al MISMO intent en corridas repetidas. Correr ademas con
// `go test -count=100 -run T4` refuerza la deteccion de aleatoriedad de
// iteracion de map entre procesos.
func TestEngine_T4_AmbiguousRegexOrderIsStableAndDeclared(t *testing.T) {
	input := "cuanto gane vendiendo"

	first := Parse(input, fixedNow)
	if first.Tier != "regex" || first.Confidence != 1.0 {
		t.Fatalf("Parse(%q) = {Tier:%q Confidence:%v}, se esperaba resolver via T1 (regex, confidence 1.0)", input, first.Tier, first.Confidence)
	}
	if first.Intent != IntentSales && first.Intent != IntentProfit {
		t.Fatalf("Parse(%q).Intent = %s, se esperaba SALES o PROFIT segun el orden declarado en el registro", input, first.Intent)
	}

	for i := 0; i < 100; i++ {
		got := Parse(input, fixedNow)
		if got.Intent != first.Intent {
			t.Fatalf("corrida %d: Intent cambio de %s a %s — indica orden no fijo (iteracion de map) en el registro de intents", i, first.Intent, got.Intent)
		}
	}
}

// TestEngine_T5_FuzzyMandatoryCases cubre AC-5: los 4 casos obligatorios
// citados textualmente por el contrato.
func TestEngine_T5_FuzzyMandatoryCases(t *testing.T) {
	cases := []struct {
		input  string
		intent Intent
	}{
		{"kien me debe plata", IntentDebtors},
		{"cuanto bendi oy", IntentSales},
		{"ke se esta acabando", IntentLowStock},
		{"en ke gano mas", IntentTopMargin},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got := Parse(c.input, fixedNow)
			if got.Intent != c.intent {
				t.Errorf("Parse(%q).Intent = %s, want %s", c.input, got.Intent, c.intent)
			}
			if got.Tier != "fuzzy" {
				t.Errorf("Parse(%q).Tier = %q, want \"fuzzy\"", c.input, got.Tier)
			}
			if got.Confidence >= 1.0 {
				t.Errorf("Parse(%q).Confidence = %v, want < 1.0", c.input, got.Confidence)
			}
		})
	}

	// "cuanto bendi oy" ademas debe resolver periodo hoy (AC-5 lo exige
	// explicitamente: "SALES + periodo hoy").
	got := Parse("cuanto bendi oy", fixedNow)
	wantFrom := midnight(2026, 9, 23)
	wantTo := midnight(2026, 9, 24)
	if got.Period.Unresolved || !got.Period.From.Equal(wantFrom) || !got.Period.To.Equal(wantTo) {
		t.Errorf(`Parse("cuanto bendi oy").Period = [%v,%v) Unresolved=%v, want [%v,%v) Unresolved=false`,
			got.Period.From, got.Period.To, got.Period.Unresolved, wantFrom, wantTo)
	}
}

// TestEngine_T6_UnknownBelowThreshold cubre AC-6.
func TestEngine_T6_UnknownBelowThreshold(t *testing.T) {
	got := Parse("¿cómo hago un sándwich?", fixedNow)
	if got.Intent != IntentUnknown {
		t.Errorf("Parse(%q).Intent = %s, want UNKNOWN", "¿cómo hago un sándwich?", got.Intent)
	}
}

// TestEngine_T7_AmbiguousOnCloseFuzzyScores cubre AC-7. Ver nota de diseño
// (3) al inicio del archivo: esta frase asume que el banco semilla de
// SALES incluye "cuanto vendi" y el de PROFIT incluye "cuanto gane".
func TestEngine_T7_AmbiguousOnCloseFuzzyScores(t *testing.T) {
	got := Parse("cuanto bendi y gaine", fixedNow)
	if got.Intent != IntentAmbiguous {
		t.Fatalf("Parse(%q).Intent = %s (tier=%s confidence=%.2f), want AMBIGUOUS — si el banco semilla del implementer no incluye \"cuanto vendi\"/\"cuanto gane\", ajustar la frase de entrada de este test",
			"cuanto bendi y gaine", got.Intent, got.Tier, got.Confidence)
	}
	if len(got.Candidates) != 2 {
		t.Fatalf("Candidates = %v, want exactamente 2", got.Candidates)
	}
	seen := map[Intent]bool{got.Candidates[0]: true, got.Candidates[1]: true}
	if !seen[IntentSales] || !seen[IntentProfit] {
		t.Fatalf("Candidates = %v, want [SALES PROFIT] (en cualquier orden)", got.Candidates)
	}
}

// TestEngine_T16_CompareFlagIndependentOfPeriodAndIntent cubre AC-17.
func TestEngine_T16_CompareFlagIndependentOfPeriodAndIntent(t *testing.T) {
	got := Parse("cuanto vendi hoy comparado con ayer", fixedNow)
	if got.Intent != IntentSales {
		t.Errorf("Intent = %s, want SALES", got.Intent)
	}
	if !got.Compare {
		t.Errorf("Compare = false, want true")
	}
	wantFrom := midnight(2026, 9, 23)
	wantTo := midnight(2026, 9, 24)
	if got.Period.Unresolved || !got.Period.From.Equal(wantFrom) || !got.Period.To.Equal(wantTo) {
		t.Errorf("Period = [%v,%v) Unresolved=%v, want [%v,%v) Unresolved=false (periodo hoy)",
			got.Period.From, got.Period.To, got.Period.Unresolved, wantFrom, wantTo)
	}
}

// TestEngine_T17_UnknownOnOutOfDomainQuestion cubre AC-18 (mismo caso que
// T-6, doble cobertura intencional bordes+T2 segun tabla del contrato).
func TestEngine_T17_UnknownOnOutOfDomainQuestion(t *testing.T) {
	got := Parse("¿cómo hago un sándwich?", fixedNow)
	if got.Intent != IntentUnknown {
		t.Errorf("Intent = %s, want UNKNOWN", got.Intent)
	}
}

// TestEngine_T18_EmptyAndBlankInputsNeverPanic cubre AC-19.
func TestEngine_T18_EmptyAndBlankInputsNeverPanic(t *testing.T) {
	inputs := []string{"", "   ", "\t\n"}
	for _, in := range inputs {
		in := in
		t.Run("input_"+in, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Parse(%q) hizo panic: %v", in, r)
				}
			}()
			got := Parse(in, fixedNow)
			if got.Intent != IntentUnknown {
				t.Errorf("Parse(%q).Intent = %s, want UNKNOWN", in, got.Intent)
			}
		})
	}
}

// TestEngine_T19_LongInputBoundedTime cubre AC-20.
func TestEngine_T19_LongInputBoundedTime(t *testing.T) {
	input := strings.Repeat("a", 5000)
	start := time.Now()
	got := Parse(input, fixedNow)
	elapsed := time.Since(start)

	if got.Intent != IntentUnknown {
		t.Errorf("Parse(5000 chars).Intent = %s, want UNKNOWN", got.Intent)
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("Parse(5000 chars) tardo %v, se esperaba tiempo acotado (<200ms), posible backtracking catastrofico de regex", elapsed)
	}
}

// TestEngine_T20_NoForbiddenImports cubre AC-21 (revision estatica escrita
// como test Go, per instruccion del orquestador). Lee el codigo fuente del
// paquete (no lo modifica) y falla si aparece un import prohibido.
func TestEngine_T20_NoForbiddenImports(t *testing.T) {
	forbidden := []string{`"database/sql"`, `"net/http"`, `"github.com/andreyo/pos/internal/repository"`}
	files := []string{"engine.go", "normalize.go", "period.go"}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("no se pudo leer %s (paquete internal/assistant aun no implementado): %v", f, err)
		}
		content := string(data)
		for _, forb := range forbidden {
			if strings.Contains(content, forb) {
				t.Errorf("%s importa %s, prohibido por AC-21", f, forb)
			}
		}
	}
}

// TestEngine_T21_ParseIsOnlyExportedEngineFunction cubre AC-22 (revision
// estatica escrita como test Go). Cuenta funciones exportadas de nivel
// paquete en los .go de produccion; tipos exportados (Query, Intent, etc.)
// no cuentan, solo declaraciones `func NombreExportado(`.
func TestEngine_T21_ParseIsOnlyExportedEngineFunction(t *testing.T) {
	files := []string{"engine.go", "normalize.go", "period.go"}
	exportedFuncRe := regexp.MustCompile(`(?m)^func ([A-Z]\w*)\(`)
	var exported []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("no se pudo leer %s (paquete internal/assistant aun no implementado): %v", f, err)
		}
		matches := exportedFuncRe.FindAllStringSubmatch(string(data), -1)
		for _, m := range matches {
			exported = append(exported, m[1])
		}
	}
	if len(exported) != 1 || exported[0] != "Parse" {
		t.Fatalf("funciones exportadas encontradas: %v, se esperaba exactamente [Parse] (AC-22)", exported)
	}
}

// TestEngine_T22_DeterministicAcrossRepeatedCalls cubre AC-23: el mismo
// input, corrido muchas veces (y reforzado externamente con
// `go test -count=50`), debe producir SIEMPRE el mismo resultado completo
// — detecta dependencia de orden de iteracion de map.
func TestEngine_T22_DeterministicAcrossRepeatedCalls(t *testing.T) {
	input := "cuanto vendi hoy"
	first := Parse(input, fixedNow)
	for i := 0; i < 200; i++ {
		got := Parse(input, fixedNow)
		if !queriesEqual(got, first) {
			t.Fatalf("corrida %d: resultado cambio de %+v a %+v — indica dependencia de orden de iteracion de map", i, first, got)
		}
	}
}

func queriesEqual(a, b Query) bool {
	if a.Intent != b.Intent || a.Compare != b.Compare || a.Entity != b.Entity ||
		a.Confidence != b.Confidence || a.Tier != b.Tier {
		return false
	}
	if !a.Period.From.Equal(b.Period.From) || !a.Period.To.Equal(b.Period.To) || a.Period.Unresolved != b.Period.Unresolved {
		return false
	}
	if len(a.Candidates) != len(b.Candidates) {
		return false
	}
	for i := range a.Candidates {
		if a.Candidates[i] != b.Candidates[i] {
			return false
		}
	}
	return true
}

// TestEngine_T23_BankHasAtLeast60Phrases cubre AC-24 explicitamente (fila
// T-23 de la tabla, separada de T-3 en el contrato).
func TestEngine_T23_BankHasAtLeast60Phrases(t *testing.T) {
	if len(engineT3Cases) < 60 {
		t.Fatalf("len(engineT3Cases) = %d, want >= 60", len(engineT3Cases))
	}
}

// TestEngine_T24_NoOverlapBetweenIntentBanks cubre AC-25: ninguna frase del
// banco de un intent debe resolver a un intent distinto del suyo. Ver nota
// de diseño (2) al inicio del archivo sobre por que reutiliza engineT3Cases.
func TestEngine_T24_NoOverlapBetweenIntentBanks(t *testing.T) {
	for _, tc := range engineT3Cases {
		got := Parse(tc.input, fixedNow)
		if got.Intent != tc.intent {
			t.Errorf("%q pertenece al banco de %s pero resolvio a %s (solapamiento entre bancos, viola AC-25)", tc.input, tc.intent, got.Intent)
		}
	}
}

// TestEngine_T26_LevenshteinArithmeticAndLengthGuard cubre el requisito de
// DoD (linea 322 del sprint) sobre la funcion `levenshtein`, nombrada
// explicitamente asi por el propio contrato (AC-5, §2).
func TestEngine_T26_LevenshteinArithmeticAndLengthGuard(t *testing.T) {
	if got := levenshtein("vendi", "bendi"); got != 1 {
		t.Errorf("levenshtein(vendi,bendi) = %d, want 1", got)
	}
	if got := levenshtein("hoy", "doy"); got != 1 {
		t.Errorf("levenshtein(hoy,doy) = %d, want 1", got)
	}

	// "hoy" no debe disparar fuzzy de intent porque len("hoy") < 4, aunque
	// levenshtein(hoy,doy)==1. Una consulta que solo contiene "doy" no
	// pertenece a ningun vocabulario de intent, asi que debe caer en
	// UNKNOWN (no en SALES via un supuesto match fuzzy de "hoy").
	got := Parse("doy", fixedNow)
	if got.Intent != IntentUnknown {
		t.Errorf(`Parse("doy").Intent = %s, want UNKNOWN (hoy/doy no debe activar fuzzy de intent por longitud < 4)`, got.Intent)
	}
}

// TestEngine_T27_ExactSynonymOverridesLevenshteinDistance2 cubre AC-5: la
// lista de sinonimos exactos tiene prioridad y resuelve casos que el fuzzy
// puro (levenshtein<=1) no alcanzaria por si solo.
func TestEngine_T27_ExactSynonymOverridesLevenshteinDistance2(t *testing.T) {
	if d := levenshtein("kien", "quien"); d != 2 {
		t.Fatalf("precondicion invalida: levenshtein(kien,quien) = %d, se esperaba 2 (si cambia, revisar el diseño del sinonimo exacto)", d)
	}

	got := Parse("kien me debe plata", fixedNow)
	if got.Intent != IntentDebtors {
		t.Fatalf(`Parse("kien me debe plata").Intent = %s, want DEBTORS via sinonimo exacto kien->quien`, got.Intent)
	}
	if got.Tier != "fuzzy" || got.Confidence >= 1.0 {
		t.Fatalf("Tier=%q Confidence=%v, want Tier=fuzzy Confidence<1.0", got.Tier, got.Confidence)
	}
}
