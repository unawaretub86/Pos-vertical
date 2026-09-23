package assistant

// Tests del extractor de periodo — contrato US-060 §3 "Periodo"
// (AC-8 a AC-16). Se prueban a través de Parse(q, now).Period porque el
// contrato solo exige que Parse sea la unica funcion exportada del motor
// (AC-22); el extractor de periodo es un detalle interno de period.go que
// no necesita su propio simbolo exportado para ser observable.
//
// fixedNow (definido aqui, usado tambien por engine_test.go dentro del
// mismo paquete) es un miercoles: 2026-09-23 15:30. Lunes de esta semana:
// 2026-09-21. Lunes de la semana previa: 2026-09-14. Mes previo: agosto
// 2026 (para AC-13/T-30). Verificado con `date -d 2026-09-23 +%A` ->
// Wednesday.
//
// GAP RESUELTO (2026-09-23, confirmado por Andrey/PO): AC-13 ("mes
// pasado") y AC-14 ("este año"/"este ano") estaban en §3 del contrato pero
// no tenian fila en la tabla "Tests requeridos" de §4. Se agregaron T-30
// (AC-13) y T-31 (AC-14) siguiendo el mismo patron que T-11/T-12. Ver
// tambien la actualizacion de la tabla §4 en changes/US-060/contract.md.

import (
	"testing"
	"time"
)

var fixedNow = time.Date(2026, time.September, 23, 15, 30, 0, 0, time.Local)

func midnight(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

func assertPeriod(t *testing.T, label string, got Period, wantFrom, wantTo time.Time, wantUnresolved bool) {
	t.Helper()
	if got.Unresolved != wantUnresolved {
		t.Errorf("%s: Unresolved = %v, want %v", label, got.Unresolved, wantUnresolved)
	}
	if wantUnresolved {
		return
	}
	if !got.From.Equal(wantFrom) {
		t.Errorf("%s: Period.From = %v, want %v", label, got.From, wantFrom)
	}
	if !got.To.Equal(wantTo) {
		t.Errorf("%s: Period.To = %v, want %v", label, got.To, wantTo)
	}
}

// TestPeriod_T8_NoTimeWordDefaultsToToday cubre AC-8.
func TestPeriod_T8_NoTimeWordDefaultsToToday(t *testing.T) {
	got := Parse("cuánto vendí", fixedNow).Period
	assertPeriod(t, "cuánto vendí", got, midnight(2026, 9, 23), midnight(2026, 9, 24), false)
}

// TestPeriod_T9_Yesterday cubre AC-9.
func TestPeriod_T9_Yesterday(t *testing.T) {
	got := Parse("cuánto vendí ayer", fixedNow).Period
	assertPeriod(t, "cuánto vendí ayer", got, midnight(2026, 9, 22), midnight(2026, 9, 23), false)
}

// TestPeriod_T10_ThisWeekStartsMonday cubre AC-10: [lunes 00:00, ahora).
func TestPeriod_T10_ThisWeekStartsMonday(t *testing.T) {
	got := Parse("cuánto vendí esta semana", fixedNow).Period
	if got.Unresolved {
		t.Fatalf("esta semana: Unresolved = true, want false")
	}
	wantFrom := midnight(2026, 9, 21) // lunes de esta semana
	if !got.From.Equal(wantFrom) {
		t.Errorf("esta semana: From = %v, want %v (lunes)", got.From, wantFrom)
	}
	if !got.To.Equal(fixedNow) {
		t.Errorf("esta semana: To = %v, want %v (ahora)", got.To, fixedNow)
	}
}

// TestPeriod_T11_LastWeekMondayToMonday cubre AC-11.
func TestPeriod_T11_LastWeekMondayToMonday(t *testing.T) {
	got := Parse("cuánto vendí la semana pasada", fixedNow).Period
	assertPeriod(t, "semana pasada", got, midnight(2026, 9, 14), midnight(2026, 9, 21), false)
}

// TestPeriod_T12_ThisMonthFromDay1 cubre AC-12: [día 1 00:00, ahora).
func TestPeriod_T12_ThisMonthFromDay1(t *testing.T) {
	got := Parse("cuánto vendí este mes", fixedNow).Period
	if got.Unresolved {
		t.Fatalf("este mes: Unresolved = true, want false")
	}
	wantFrom := midnight(2026, 9, 1)
	if !got.From.Equal(wantFrom) {
		t.Errorf("este mes: From = %v, want %v", got.From, wantFrom)
	}
	if !got.To.Equal(fixedNow) {
		t.Errorf("este mes: To = %v, want %v (ahora)", got.To, fixedNow)
	}
}

// TestPeriod_T13_TypoAllerResolvesToYesterdayViaSynonym cubre AC-9 via el
// sinonimo exacto "aller"->"ayer" (decision GATE1, distancia Levenshtein
// real = 2, fuera del umbral <=1 general).
func TestPeriod_T13_TypoAllerResolvesToYesterdayViaSynonym(t *testing.T) {
	got := Parse("cuanto vendi aller", fixedNow).Period
	assertPeriod(t, "cuanto vendi aller", got, midnight(2026, 9, 22), midnight(2026, 9, 23), false)
}

// TestPeriod_T14_WeekdayWordIsUnresolvedNeverToday cubre AC-15/AC-16:
// "juebes" (variante de error de dedo de "jueves") matchea el vocabulario
// fijo de dias de la semana -> Unresolved = true, JAMAS periodo hoy.
func TestPeriod_T14_WeekdayWordIsUnresolvedNeverToday(t *testing.T) {
	got := Parse("cuanto vendi el juebes", fixedNow).Period
	if !got.Unresolved {
		t.Fatalf("juebes: Unresolved = false, want true (nunca debe caer en 'hoy' por defecto)")
	}
}

// TestPeriod_T15_AllerVsJuebesVsNoTimeWordSameTable cubre AC-8, AC-9 y
// AC-15 en una sola tabla, tal como pide el DoD (linea 325 del sprint):
// el bug tipico es confundir estos tres casos entre si.
func TestPeriod_T15_AllerVsJuebesVsNoTimeWordSameTable(t *testing.T) {
	cases := []struct {
		label      string
		input      string
		unresolved bool
		from, to   time.Time
	}{
		{"aller -> ayer", "cuanto vendi aller", false, midnight(2026, 9, 22), midnight(2026, 9, 23)},
		{"juebes -> unresolved", "cuanto vendi el juebes", true, time.Time{}, time.Time{}},
		{"sin palabra de tiempo -> hoy", "cuanto vendi", false, midnight(2026, 9, 23), midnight(2026, 9, 24)},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			got := Parse(c.input, fixedNow).Period
			if got.Unresolved != c.unresolved {
				t.Fatalf("%q: Unresolved = %v, want %v", c.input, got.Unresolved, c.unresolved)
			}
			if c.unresolved {
				return
			}
			if !got.From.Equal(c.from) || !got.To.Equal(c.to) {
				t.Errorf("%q: Period = [%v,%v), want [%v,%v)", c.input, got.From, got.To, c.from, c.to)
			}
		})
	}
}

// TestPeriod_T28_AllSevenWeekdaysAreUnresolved cubre AC-15 para los 7 dias,
// no solo "jueves": ninguno debe resolver a un periodo por defecto.
func TestPeriod_T28_AllSevenWeekdaysAreUnresolved(t *testing.T) {
	days := []string{"lunes", "martes", "miercoles", "jueves", "viernes", "sabado", "domingo"}
	for _, d := range days {
		input := "cuanto vendi el " + d
		t.Run(d, func(t *testing.T) {
			got := Parse(input, fixedNow).Period
			if !got.Unresolved {
				t.Fatalf("%q: Unresolved = false, want true", input)
			}
		})
	}
}

// TestPeriod_T29_UnrelatedWordDoesNotTriggerPeriodLogic cubre AC-15: una
// palabra ajena a ambos vocabularios (temporal y dias de la semana) no
// dispara nada relacionado con periodo -> periodo hoy por defecto,
// Unresolved = false.
func TestPeriod_T29_UnrelatedWordDoesNotTriggerPeriodLogic(t *testing.T) {
	got := Parse("cuanto vendi de queso", fixedNow).Period
	if got.Unresolved {
		t.Fatalf(`"cuanto vendi de queso": Unresolved = true, want false (queso no es candidata a tiempo)`)
	}
	if !got.From.Equal(midnight(2026, 9, 23)) || !got.To.Equal(midnight(2026, 9, 24)) {
		t.Errorf("expected default hoy period, got [%v,%v)", got.From, got.To)
	}
}

// TestPeriod_T30_LastMonthDay1ToDay1 cubre AC-13 (mes pasado): [día 1 00:00
// del mes previo, día 1 00:00 del mes actual). Con fixedNow en septiembre
// 2026, el mes previo es agosto.
func TestPeriod_T30_LastMonthDay1ToDay1(t *testing.T) {
	got := Parse("cuánto vendí el mes pasado", fixedNow).Period
	assertPeriod(t, "mes pasado", got, midnight(2026, 8, 1), midnight(2026, 9, 1), false)
}

// TestPeriod_T31_ThisYearFromJan1WithAndWithoutEnie cubre AC-14: "este
// año" y su variante sin eñe "este ano" (comun en teclados sin ñ, distinto
// del caso de tildes porque la ñ nunca se toca en normalizacion, AC-2) DEBEN
// resolver el mismo periodo: [1 de enero 00:00 del año actual, ahora).
func TestPeriod_T31_ThisYearFromJan1WithAndWithoutEnie(t *testing.T) {
	cases := []struct {
		label string
		input string
	}{
		{"con eñe (año)", "cuánto vendí este año"},
		{"sin eñe (ano)", "cuánto vendí este ano"},
	}
	wantFrom := midnight(2026, 1, 1)
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			got := Parse(c.input, fixedNow).Period
			if got.Unresolved {
				t.Fatalf("%q: Unresolved = true, want false", c.input)
			}
			if !got.From.Equal(wantFrom) {
				t.Errorf("%q: From = %v, want %v (1 de enero)", c.input, got.From, wantFrom)
			}
			if !got.To.Equal(fixedNow) {
				t.Errorf("%q: To = %v, want %v (ahora)", c.input, got.To, fixedNow)
			}
		})
	}
}
