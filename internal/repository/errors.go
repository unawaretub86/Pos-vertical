package repository

import "errors"

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")

// ErrRequiresCreditNote is returned when voiding a sale that has an accepted electronic document.
type ErrRequiresCreditNote struct {
	DocumentID string
	CUDE       string
}

func (e *ErrRequiresCreditNote) Error() string {
	return "la venta tiene un documento electrónico aceptado; se requiere nota crédito para anular"
}
