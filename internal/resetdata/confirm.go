package resetdata

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Confirm reads one line from r and accepts exactly "BORRAR <sales>".
// Empty or closed input is a refusal.
func Confirm(r io.Reader, sales int) error {
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: leer la confirmación: %w", ErrConfirmation, err)
	}
	want := fmt.Sprintf("BORRAR %d", sales)
	if strings.TrimRight(line, "\r\n") != want {
		return fmt.Errorf("%w: se esperaba %q", ErrConfirmation, want)
	}
	return nil
}
