package resetdata

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/andreyo/pos/internal/database"
	_ "modernc.org/sqlite"
)

// queryer is satisfied by *sql.DB and *sql.Conn, so guards run both on the
// pool and inside the exclusive transaction.
type queryer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Open opens the database the way cmd/reset-password does: raw sql.Open, no
// migrations. A missing file is a usage error and nothing is created.
func Open(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: no existe la base de datos %s", ErrUsage, path)
		}
		return nil, fmt.Errorf("%w: %s: %w", ErrUsage, path, err)
	}
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("%w: abrir %s: %w", ErrDatabase, path, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("%w: abrir %s: %w", ErrDatabase, path, err)
	}
	return db, nil
}

// CheckSchemaVersion refuses when schema_migrations differs from the binary's
// latest migration. The tool never migrates.
func CheckSchemaVersion(ctx context.Context, db queryer) error {
	var v sql.NullInt64
	if err := db.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&v); err != nil {
		return fmt.Errorf("%w: leer schema_migrations: %w", ErrDatabase, err)
	}
	if int(v.Int64) != database.SchemaVersion() {
		return fmt.Errorf("%w (base: %d, herramienta: %d)", ErrSchemaVersion, v.Int64, database.SchemaVersion())
	}
	return nil
}

// portInUse reports whether something accepts TCP connections on 127.0.0.1:port.
func portInUse(port int) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// CheckPort refuses when the POS port answers (the POS is running).
func CheckPort(port int) error {
	if portInUse(port) {
		return fmt.Errorf("%w (puerto %d)", ErrPOSRunning, port)
	}
	return nil
}

// fiscalCounts returns the number of electronic documents per blocking status.
func fiscalCounts(ctx context.Context, q queryer) (map[string]int, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT status, COUNT(*) FROM electronic_documents WHERE status IN ('accepted', 'draft', 'pending') GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("%w: leer electronic_documents: %w", ErrDatabase, err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var s string
		var n int
		if err := rows.Scan(&s, &n); err != nil {
			return nil, fmt.Errorf("%w: leer electronic_documents: %w", ErrDatabase, err)
		}
		out[s] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: leer electronic_documents: %w", ErrDatabase, err)
	}
	return out, nil
}

// fiscalOrder is the display order of the blocking statuses.
var fiscalOrder = []string{"accepted", "draft", "pending"}

// firstActiveAdmin returns the lowest-id active admin, or ErrNoAdmin.
func firstActiveAdmin(ctx context.Context, q queryer) (int64, error) {
	var id int64
	err := q.QueryRowContext(ctx, `SELECT id FROM users WHERE role = 'admin' AND active = 1 ORDER BY id LIMIT 1`).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNoAdmin
	}
	if err != nil {
		return 0, fmt.Errorf("%w: leer usuarios: %w", ErrDatabase, err)
	}
	return id, nil
}

// checkGuards runs the data guards: active admin, fiscal documents, second run.
func checkGuards(ctx context.Context, q queryer, opts Options) error {
	if _, err := firstActiveAdmin(ctx, q); err != nil {
		return err
	}
	if !opts.DiscardFiscal {
		counts, err := fiscalCounts(ctx, q)
		if err != nil {
			return err
		}
		var parts []string
		for _, s := range fiscalOrder {
			if counts[s] > 0 {
				parts = append(parts, fmt.Sprintf("%s %d", s, counts[s]))
			}
		}
		if len(parts) > 0 {
			return fmt.Errorf("%w: %s (use --discard-fiscal-documents para borrarlos; --force no lo permite)",
				ErrFiscal, strings.Join(parts, ", "))
		}
	}
	if !opts.Force {
		var n int
		if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_log WHERE action = 'data_reset'`).Scan(&n); err != nil {
			return fmt.Errorf("%w: leer audit_log: %w", ErrDatabase, err)
		}
		if n > 0 {
			return ErrAlreadyReset
		}
	}
	return nil
}

// CheckGuards runs the guards that need no lock: active admin, fiscal
// documents, second run. Apply repeats them inside its transaction.
func CheckGuards(ctx context.Context, db queryer, opts Options) error {
	return checkGuards(ctx, db, opts)
}
