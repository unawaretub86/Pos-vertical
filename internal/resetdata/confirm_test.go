package resetdata

import (
	"errors"
	"strings"
	"testing"
)

// T-11: only the exact text "BORRAR <N>" proceeds; the database is never read or written.
func TestConfirm_T11_TypedConfirmation(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		sales   int
		proceed bool
	}{
		{"exact", "BORRAR 4\n", 4, true},
		{"lowercase", "borrar 4\n", 4, false},
		{"wrong number", "BORRAR 5\n", 4, false},
		{"empty stdin", "", 4, false},
		{"si", "si\n", 4, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := Confirm(strings.NewReader(c.input), c.sales)
			if c.proceed {
				if err != nil {
					t.Fatalf("Confirm(%q, %d) = %v, want nil", c.input, c.sales, err)
				}
				return
			}
			if !errors.Is(err, ErrConfirmation) || !errors.Is(err, ErrRefused) {
				t.Fatalf("Confirm(%q, %d) = %v, want ErrConfirmation (refused)", c.input, c.sales, err)
			}
			if ExitCode(err) != 1 {
				t.Errorf("ExitCode = %d, want 1", ExitCode(err))
			}
		})
	}
}
