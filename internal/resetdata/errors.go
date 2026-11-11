// Package resetdata implements the go-live reset: it removes dry-run
// transactions and keeps master data (US-093). Used only by cmd/reset-data.
package resetdata

import "errors"

// Sentinels map to the exit codes of the tool.
var (
	// ErrRefused: a guard stopped the run before any change (exit 1).
	ErrRefused = errors.New("operación rechazada")
	// ErrUsage: bad flags or a missing database file (exit 2).
	ErrUsage = errors.New("uso incorrecto")
	// ErrBackup: the backup could not be created or verified (exit 3).
	ErrBackup = errors.New("falló el respaldo")
	// ErrDatabase: a database error, rolled back (exit 4).
	ErrDatabase = errors.New("error de base de datos")
)

// refusal is a refusal sentinel with its own text; errors.Is(x, ErrRefused) holds.
type refusal struct{ msg string }

func (r *refusal) Error() string { return r.msg }
func (r *refusal) Unwrap() error { return ErrRefused }

var (
	ErrAlreadyReset  error = &refusal{"ya se ejecutó un reinicio de datos antes (use --force para repetirlo)"}
	ErrFiscal        error = &refusal{"hay documentos electrónicos que no se pueden borrar"}
	ErrNoAdmin       error = &refusal{"no hay un administrador activo; el registro de auditoría lo necesita"}
	ErrConfirmation  error = &refusal{"confirmación incorrecta"}
	ErrPOSRunning    error = &refusal{"el POS está abierto. Ciérrelo y vuelva a intentarlo"}
	ErrLocked        error = &refusal{"la base de datos está en uso por otro programa. Cierre el POS y vuelva a intentarlo"}
	ErrSchemaVersion error = &refusal{"la versión de la base de datos no coincide con la de esta herramienta; abra pos.exe una vez para migrar"}
)

// ExitCode maps an error to the process exit code: 0 nil, 1 refusal, 2 usage,
// 3 backup, 4 database (and anything unknown).
func ExitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, ErrRefused):
		return 1
	case errors.Is(err, ErrUsage):
		return 2
	case errors.Is(err, ErrBackup):
		return 3
	default:
		return 4
	}
}
