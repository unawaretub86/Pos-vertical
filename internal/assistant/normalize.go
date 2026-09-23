// Package assistant translates a free-form Spanish question typed by a
// store owner into a structured Query{Intent, Period}. It is pure text
// processing: no database, no HTTP, no repositories (AC-21). US-061 will
// wire this engine to internal/repository/metrics.go later.
package assistant

import "strings"

// accentReplacer strips tildes and diéresis by hand (á->a, é->e, í->i,
// ó->o, ú->u, ü->u). The "ñ" is deliberately excluded: it is a letter of
// its own in Spanish, not an accented vowel, and AC-2 forbids touching it
// (golang.org/x/text/unicode/norm is not in go.mod and would be a new
// dependency, forbidden by the contract §7).
var accentReplacer = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u",
)

// punctReplacer removes the punctuation signs listed explicitly by AC-1.
var punctReplacer = strings.NewReplacer(
	"¿", "", "?", "", "¡", "", "!", "", ",", "", ".", "", ";", "",
)

// stopwords are stripped from the normalized form per AC-1. Membership
// lookup only (never iterated), so this map does not violate the
// map-iteration-is-nondeterministic rule from CLAUDE.md.
var stopwords = map[string]bool{
	"de": true, "la": true, "el": true, "los": true, "las": true,
	"mi": true, "me": true, "un": true, "una": true, "que": true,
	"en": true, "por": true, "y": true,
}

// stripAccentsLowerPunct lowercases, strips tildes/diéresis (keeping ñ) and
// the punctuation signs from AC-1, then collapses/trims whitespace. It does
// NOT remove stopwords — engine.go reuses it (via compare-phrase detection)
// on cases where a stopword like "que" is part of the signal being looked
// for (AC-17: "que ayer", "que el mes pasado").
func stripAccentsLowerPunct(s string) string {
	s = strings.ToLower(s)
	s = accentReplacer.Replace(s)
	s = punctReplacer.Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

// normalize produces the canonical form used by both T1 (regex) and T2
// (fuzzy) matching: lowercase, no tildes/diéresis (ñ preserved, AC-2), no
// ¿ ? ¡ ! , . ; , no stopwords (de, la, el, los, las, mi, me, un, una, que,
// en, por, y), single spaces, trimmed (AC-1).
func normalize(s string) string {
	s = stripAccentsLowerPunct(s)
	fields := strings.Fields(s)
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if stopwords[f] {
			continue
		}
		out = append(out, f)
	}
	return strings.Join(out, " ")
}
