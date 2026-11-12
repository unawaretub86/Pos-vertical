package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// maxBackups is how many daily backup files are kept in the folder written.
const maxBackups = 30

// BackupSource tells the scheduler which folder to write to. BackupDir may
// block (it reads the database); an empty string means the default folder,
// next to the database file.
type BackupSource interface {
	BackupDir() string
}

// BackupRecorder is optionally implemented by a BackupSource to persist the
// result of each attempt (at is the attempt time).
type BackupRecorder interface {
	RecordBackup(at time.Time, ok, fellBack bool, errMsg, effectiveDir, file string) error
}

// BackupResult is the outcome of one backup attempt.
type BackupResult struct {
	OK           bool
	FellBack     bool
	Error        string
	EffectiveDir string
	File         string
}

// StartBackupScheduler runs a backup immediately and then every 24 hours.
// It returns at once: the backup runs in a background goroutine and never
// causes the process to exit.
func (d *DB) StartBackupScheduler(src BackupSource) {
	go func() {
		d.RunBackup(src)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			d.RunBackup(src)
		}
	}()
}

// RunBackup takes a backup and records the result through src when src
// implements BackupRecorder.
func (d *DB) RunBackup(src BackupSource) BackupResult {
	res := d.runBackup(src)
	if rec, ok := src.(BackupRecorder); ok {
		if err := rec.RecordBackup(d.now(), res.OK, res.FellBack, res.Error, res.EffectiveDir, res.File); err != nil {
			log.Printf("[backup] could not record result: %v", err)
		}
	}
	return res
}

// DefaultBackupDir is the folder of the database file (absolute when possible).
func (d *DB) DefaultBackupDir() string {
	if abs, err := filepath.Abs(filepath.Dir(d.path)); err == nil {
		return abs
	}
	return filepath.Dir(d.path)
}

// CanWriteDir reports whether a file can be created in dir.
func CanWriteDir(dir string) error {
	f, err := os.CreateTemp(dir, ".pos-write-test-*")
	if err != nil {
		return err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		os.Remove(name)
		return err
	}
	return os.Remove(name)
}

func (d *DB) runBackup(src BackupSource) BackupResult {
	var res BackupResult
	dir := d.DefaultBackupDir()

	configured := ""
	if src != nil {
		configured = strings.TrimSpace(src.BackupDir())
	}
	fallbackMsg := ""
	if configured != "" {
		if err := d.checkBackupDir(configured); err != nil {
			res.FellBack = true
			fallbackMsg = fmt.Sprintf("%s: %v", configured, err)
			log.Printf("[backup] configured folder unusable, using default: %s", fallbackMsg)
		} else {
			dir = configured
		}
	}
	res.EffectiveDir = dir

	fail := func(err error) BackupResult {
		res.OK = false
		res.Error = err.Error()
		if fallbackMsg != "" {
			res.Error = fallbackMsg + "; " + res.Error
		}
		log.Printf("[backup] failed: %s", res.Error)
		return res
	}

	base := filepath.Base(d.path)
	date := d.now().In(time.Local).Format("20060102")
	dest := filepath.Join(dir, base+".backup-"+date)
	tmp := dest + ".tmp"

	if err := d.writeVerifiedCopy(tmp); err != nil {
		return fail(err)
	}
	if err := os.Rename(tmp, dest); err != nil {
		removeIfFile(tmp)
		return fail(fmt.Errorf("rename backup: %w", err))
	}
	log.Printf("[backup] created %s", dest)

	res.OK = true
	res.File = dest
	res.Error = fallbackMsg
	if err := pruneBackups(dir, base); err != nil {
		log.Printf("[backup] prune failed: %v", err)
	}
	return res
}

func (d *DB) checkBackupDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.New("La carpeta no existe")
		}
		return fmt.Errorf("No se puede leer la carpeta: %w", err)
	}
	if !info.IsDir() {
		return errors.New("La ruta no es una carpeta")
	}
	if err := d.canWrite(dir); err != nil {
		return fmt.Errorf("No se puede escribir en la carpeta: %w", err)
	}
	return nil
}

// writeVerifiedCopy writes a VACUUM INTO copy to tmp and verifies it. A copy
// that fails only because live row counts moved while it ran is retried once.
// On any failure tmp is removed (when it is a regular file).
func (d *DB) writeVerifiedCopy(tmp string) error {
	removeIfFile(tmp) // stale file from an interrupted run; a directory is left alone and fails below
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		before, err := d.liveCounts()
		if err != nil {
			return fmt.Errorf("count rows: %w", err)
		}
		if err := d.checkpoint(); err != nil {
			log.Printf("[backup] checkpoint failed: %v", err)
		}
		if _, err := d.Exec("VACUUM INTO ?", tmp); err != nil {
			removeIfFile(tmp)
			return fmt.Errorf("VACUUM INTO: %w", err)
		}
		retry, err := d.verifyCopyCounts(tmp, before)
		if err == nil {
			return nil
		}
		removeIfFile(tmp)
		lastErr = fmt.Errorf("verify backup: %w", err)
		if !retry {
			break
		}
	}
	return lastErr
}

func removeIfFile(path string) {
	if info, err := os.Lstat(path); err == nil && !info.IsDir() {
		_ = os.Remove(path)
	}
}

func (d *DB) checkpoint() error {
	_, err := d.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return err
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// liveCounts returns the row count of every user table of the live database.
func (d *DB) liveCounts() (map[string]int, error) {
	rows, err := d.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, err
	}
	var tables []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return nil, err
		}
		tables = append(tables, n)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	out := make(map[string]int, len(tables))
	for _, t := range tables {
		var n int
		if err := d.QueryRow(`SELECT COUNT(*) FROM ` + quoteIdent(t)).Scan(&n); err != nil {
			return nil, fmt.Errorf("count %s: %w", t, err)
		}
		out[t] = n
	}
	return out, nil
}

// verifyBackup checks a copy against the live database assuming it is quiet
// (live counts before and after are the same).
func (d *DB) verifyBackup(path string) error {
	before, err := d.liveCounts()
	if err != nil {
		return fmt.Errorf("count rows: %w", err)
	}
	_, err = d.verifyCopyCounts(path, before)
	return err
}

// verifyCopyCounts opens the copy, requires integrity_check = ok and, for every
// user table, min(before, after) <= copy <= max(before, after), where before is
// the live count taken ahead of the copy and after is read here. retry is true
// only when a table failed while its live count changed during the run.
func (d *DB) verifyCopyCounts(path string, before map[string]int) (retry bool, err error) {
	after, err := d.liveCounts()
	if err != nil {
		return false, fmt.Errorf("count rows: %w", err)
	}
	c, err := sql.Open("sqlite", path)
	if err != nil {
		return false, fmt.Errorf("open copy: %w", err)
	}
	defer c.Close()
	c.SetMaxOpenConns(1)

	var ic string
	if err := c.QueryRow(`PRAGMA integrity_check`).Scan(&ic); err != nil {
		return false, fmt.Errorf("integrity_check: %w", err)
	}
	if ic != "ok" {
		return false, fmt.Errorf("integrity_check: %s", ic)
	}

	tables := make([]string, 0, len(after))
	for t := range after {
		tables = append(tables, t)
	}
	sort.Strings(tables)
	var problems []string
	for _, t := range tables {
		var n int
		if err := c.QueryRow(`SELECT COUNT(*) FROM ` + quoteIdent(t)).Scan(&n); err != nil {
			return false, fmt.Errorf("count %s in copy: %w", t, err)
		}
		lo, hi := after[t], after[t]
		if b, ok := before[t]; ok {
			lo, hi = min(b, after[t]), max(b, after[t])
		}
		if n < lo || n > hi {
			problems = append(problems, fmt.Sprintf("%s has %d rows, live has %d..%d", t, n, lo, hi))
			if lo != hi {
				retry = true
			}
		}
	}
	if len(problems) > 0 {
		return retry, errors.New(strings.Join(problems, "; "))
	}
	return false, nil
}

// isBackupName reports whether name is exactly "<base>.backup-" + 8 digits.
// It is false for <base>.pre-migration-* and <base>.pre-reset-* files.
func isBackupName(name, base string) bool {
	prefix := base + ".backup-"
	if len(name) != len(prefix)+8 || !strings.HasPrefix(name, prefix) {
		return false
	}
	for _, r := range name[len(prefix):] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// pruneBackups keeps the maxBackups newest strict-name backups of base in dir.
// It lists the directory (no glob, so [ * ? in the path are harmless) and
// never touches any other file, such as <base>.pre-migration-* or <base>.pre-reset-*.
func pruneBackups(dir, base string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("list %s: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && isBackupName(e.Name(), base) {
			names = append(names, e.Name())
		}
	}
	if len(names) <= maxBackups {
		return nil
	}
	sort.Strings(names) // lexicographic = chronological for YYYYMMDD
	for _, n := range names[:len(names)-maxBackups] {
		f := filepath.Join(dir, n)
		if err := os.Remove(f); err != nil {
			log.Printf("[backup] could not remove old backup %s: %v", f, err)
		} else {
			log.Printf("[backup] removed old backup %s", f)
		}
	}
	return nil
}
