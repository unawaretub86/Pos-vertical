package assistant

import "time"

// Period is a semi-open date range [From, To) resolved from a query, or a
// marker that the user mentioned a time reference the engine could not
// resolve (Unresolved=true, AC-15/AC-16). Label is a human-readable Spanish
// name of the resolved range (US-061 AC-9), one of "Hoy", "Ayer", "Esta
// semana", "Semana pasada", "Este mes", "Mes pasado" or "Este año". It is
// the zero value ("") when Unresolved is true, since there is nothing to
// name.
type Period struct {
	From       time.Time
	To         time.Time
	Unresolved bool
	Label      string
}

// temporalWords is the vocabulary that DOES resolve to a concrete period
// (contract §2/§8, decision GATE 1).
var temporalWords = []string{
	"hoy", "ayer", "semana", "mes", "año", "ano", "pasada", "pasado", "esta", "este",
}

// weekdayWords is a vocabulary kept separate from temporalWords on purpose:
// a weekday is a "candidata a tiempo" the engine cannot resolve to a
// concrete date without more context, so it must force Unresolved=true and
// never fall back to "hoy" (AC-15/AC-16, decision GATE 1 §2).
var weekdayWords = []string{
	"lunes", "martes", "miercoles", "jueves", "viernes", "sabado", "domingo",
}

// periodSynonyms covers typo pairs whose Levenshtein distance exceeds the
// general <=1 fallback (decision GATE 1, §2 del contrato). "aller"->"ayer"
// has real distance 2 (verified in contract.md §2).
var periodSynonyms = map[string]string{
	"aller": "ayer",
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func startOfWeek(t time.Time) time.Time {
	day := startOfDay(t)
	wd := int(day.Weekday())
	if wd == 0 { // Sunday in time.Weekday is 0; ISO/Spanish week starts Monday.
		wd = 7
	}
	return day.AddDate(0, 0, -(wd - 1))
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func startOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, t.Location())
}

// wordMatches implements coincide(t,s) for period vocabulary: exact ->
// synonym exacto -> levenshtein<=1. Deliberately WITHOUT the len(t)>=4
// guard used for intent fuzzy matching: the discrepancy table in
// contract.md §2 explicitly validates "oy" (len 2) resolving to "hoy" via
// levenshtein.
func wordMatches(token, vocabWord string, synonyms map[string]string) bool {
	if token == vocabWord {
		return true
	}
	if syn, ok := synonyms[token]; ok && syn == vocabWord {
		return true
	}
	if len(token) > maxFuzzyTokenLen {
		return false
	}
	return levenshtein(token, vocabWord) <= 1
}

func tokensContainWord(tokens []string, vocabWord string, synonyms map[string]string) bool {
	for _, tok := range tokens {
		if wordMatches(tok, vocabWord, synonyms) {
			return true
		}
	}
	return false
}

func tokensContainAny(tokens []string, vocabWords []string, synonyms map[string]string) bool {
	for _, w := range vocabWords {
		if tokensContainWord(tokens, w, synonyms) {
			return true
		}
	}
	return false
}

// tokensContainWeekday reports whether any token matches the fixed weekday
// vocabulary (exact or levenshtein<=1, e.g. "juebes"~"jueves").
func tokensContainWeekday(tokens []string) bool {
	for _, tok := range tokens {
		for _, day := range weekdayWords {
			if wordMatches(tok, day, nil) {
				return true
			}
		}
	}
	return false
}

// extractPeriod resolves AC-8 to AC-16 from the already-normalized tokens
// of the query (the same tokens Parse uses for T1/T2 intent matching).
func extractPeriod(tokens []string, now time.Time) Period {
	isPast := tokensContainAny(tokens, []string{"pasada", "pasado"}, periodSynonyms)

	switch {
	case tokensContainWord(tokens, "semana", periodSynonyms) && isPast:
		monday := startOfWeek(now)
		return Period{From: monday.AddDate(0, 0, -7), To: monday, Label: "Semana pasada"}
	case tokensContainWord(tokens, "mes", periodSynonyms) && isPast:
		month := startOfMonth(now)
		return Period{From: month.AddDate(0, -1, 0), To: month, Label: "Mes pasado"}
	case tokensContainWord(tokens, "semana", periodSynonyms) && tokensContainWord(tokens, "esta", periodSynonyms):
		return Period{From: startOfWeek(now), To: now, Label: "Esta semana"}
	case tokensContainWord(tokens, "mes", periodSynonyms) && tokensContainWord(tokens, "este", periodSynonyms):
		return Period{From: startOfMonth(now), To: now, Label: "Este mes"}
	case (tokensContainWord(tokens, "año", periodSynonyms) || tokensContainWord(tokens, "ano", periodSynonyms)) &&
		tokensContainWord(tokens, "este", periodSynonyms):
		return Period{From: startOfYear(now), To: now, Label: "Este año"}
	case tokensContainWord(tokens, "ayer", periodSynonyms):
		today := startOfDay(now)
		return Period{From: today.AddDate(0, 0, -1), To: today, Label: "Ayer"}
	case tokensContainWord(tokens, "hoy", periodSynonyms):
		today := startOfDay(now)
		return Period{From: today, To: today.AddDate(0, 0, 1), Label: "Hoy"}
	case tokensContainWeekday(tokens):
		return Period{Unresolved: true}
	default:
		today := startOfDay(now)
		return Period{From: today, To: today.AddDate(0, 0, 1), Label: "Hoy"}
	}
}
