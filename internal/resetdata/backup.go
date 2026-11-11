package resetdata

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"
)

// BackupInfo describes a verified backup file.
type BackupInfo struct {
	Path   string
	Size   int64
	Counts map[string]int
}

// TableCounts returns COUNT(*) of every user table (sqlite_* excluded).
// Table names come from sqlite_master, never from the operator.
func TableCounts(ctx context.Context, db queryer) (map[string]int, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, fmt.Errorf("%w: listar tablas: %w", ErrDatabase, err)
	}
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return nil, fmt.Errorf("%w: listar tablas: %w", ErrDatabase, err)
		}
		names = append(names, n)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, fmt.Errorf("%w: listar tablas: %w", ErrDatabase, err)
	}
	out := make(map[string]int, len(names))
	for _, n := range names {
		var c int
		q := `SELECT COUNT(*) FROM "` + strings.ReplaceAll(n, `"`, `""`) + `"`
		if err := db.QueryRowContext(ctx, q).Scan(&c); err != nil {
			return nil, fmt.Errorf("%w: contar %s: %w", ErrDatabase, n, err)
		}
		out[n] = c
	}
	return out, nil
}

// Backup checkpoints the WAL, writes <dbPath>.pre-reset-YYYYMMDD-HHMMSS with
// VACUUM INTO and verifies it (integrity_check ok, same per-table counts).
// The name does not match the POS prune glob <db>.backup-*.
func Backup(ctx context.Context, db *sql.DB, dbPath string, now time.Time) (*BackupInfo, error) {
	target := dbPath + ".pre-reset-" + now.Format("20060102-150405")
	if _, err := os.Stat(target); err == nil {
		return nil, fmt.Errorf("%w: ya existe %s", ErrBackup, target)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return nil, fmt.Errorf("%w: checkpoint: %w", ErrBackup, err)
	}
	want, err := TableCounts(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBackup, err)
	}
	if _, err := db.ExecContext(ctx, `VACUUM INTO ?`, target); err != nil {
		return nil, fmt.Errorf("%w: copiar a %s: %w", ErrBackup, target, err)
	}

	cp, err := sql.Open("sqlite", target+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("%w: abrir la copia: %w", ErrBackup, err)
	}
	defer cp.Close()
	var integrity string
	if err := cp.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil {
		return nil, fmt.Errorf("%w: integrity_check: %w", ErrBackup, err)
	}
	if integrity != "ok" {
		return nil, fmt.Errorf("%w: integrity_check de la copia: %s", ErrBackup, integrity)
	}
	got, err := TableCounts(ctx, cp)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBackup, err)
	}
	if !reflect.DeepEqual(got, want) {
		return nil, fmt.Errorf("%w: los conteos de la copia no coinciden con la base", ErrBackup)
	}
	st, err := os.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBackup, err)
	}
	return &BackupInfo{Path: target, Size: st.Size(), Counts: want}, nil
}
