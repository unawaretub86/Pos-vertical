package assistant

// Tests de normalización — contrato US-060 §3 "Normalización" (AC-1, AC-2).
//
// Requiere una función `normalize(s string) string` sin exportar en
// normalize.go (mismo paquete, por eso el test puede llamarla directo sin
// pasar por Parse). AC-22 solo exige que Parse sea la única función
// EXPORTADA del motor; normalize puede (y debe, para ser testeable en
// aislamiento) quedar como símbolo interno del paquete.

import (
	"strings"
	"testing"
)

// TestNormalize_T1_ProducesCanonicalForm cubre AC-1: minúsculas, sin tildes/
// diéresis, sin signos ¿ ? ¡ ! , . ; , sin palabras vacías
// (de, la, el, los, las, mi, me, un, una, que, en, por, y), espacios
// colapsados y recortados.
func TestNormalize_T1_ProducesCanonicalForm(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "caso literal del contrato - mayusculas y signos",
			input: "¿CUÁNTO VENDÍ HOY?",
			want:  "cuanto vendi hoy",
		},
		{
			name:  "caso literal del contrato - ya normalizado",
			input: "cuanto vendi hoy",
			want:  "cuanto vendi hoy",
		},
		{
			name:  "elimina palabras vacias de y la",
			input: "¿Cuánto vendí de la tienda?",
			want:  "cuanto vendi tienda",
		},
		{
			name:  "elimina palabras vacias en y la, conserva el resto",
			input: "¡Hola! ¿Cuánto gané, hoy; en la tienda?",
			want:  "hola cuanto gane hoy tienda",
		},
		{
			name:  "colapsa espacios multiples",
			input: "cuanto   vendi     hoy",
			want:  "cuanto vendi hoy",
		},
		{
			name:  "recorta espacios en los extremos",
			input: "  cuanto vendi hoy  ",
			want:  "cuanto vendi hoy",
		},
		{
			name:  "palabra vacia 'que' se elimina",
			input: "que es lo que mas vendo",
			want:  "es lo mas vendo",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := normalize(c.input)
			if got != c.want {
				t.Errorf("normalize(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}

	// Las dos formas del caso literal del contrato deben producir EXACTAMENTE
	// la misma cadena normalizada entre sí, no solo coincidir con "want".
	a := normalize(cases[0].input)
	b := normalize(cases[1].input)
	if a != b {
		t.Errorf("normalize(%q)=%q y normalize(%q)=%q deberian ser iguales", cases[0].input, a, cases[1].input, b)
	}
}

// TestNormalize_T2_PreservesEnie cubre AC-2: la ñ nunca se elimina ni se
// sustituye, aunque el resto de tildes/diéresis sí se eliminen.
func TestNormalize_T2_PreservesEnie(t *testing.T) {
	input := "¿Cuánto vendí ESTE AÑO?"
	want := "cuanto vendi este año"

	got := normalize(input)
	if got != want {
		t.Fatalf("normalize(%q) = %q, want %q", input, got, want)
	}

	if !strings.Contains(got, "año") {
		t.Errorf("normalize(%q) = %q, se esperaba que conservara \"año\"", input, got)
	}
	if strings.Contains(got, "ano") && !strings.Contains(got, "año") {
		t.Errorf("normalize(%q) = %q, la ñ fue sustituida por n (\"ano\"), violacion de AC-2", input, got)
	}
}
