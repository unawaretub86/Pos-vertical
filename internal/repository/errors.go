package repository

import "errors"

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")

// ErrCustomerInactive is returned when a credit sale targets a customer that does not exist or was deleted (active = 0).
var ErrCustomerInactive = errors.New("el cliente no existe o está inactivo; no se le puede fiar")

// ErrRequiresCreditNote is returned when voiding a sale that has an accepted electronic document.
type ErrRequiresCreditNote struct {
	DocumentID string
	CUDE       string
}

func (e *ErrRequiresCreditNote) Error() string {
	return "la venta tiene un documento electrónico aceptado; se requiere nota crédito para anular"
}
