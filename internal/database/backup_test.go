package database

// Backup tests. Helper API assumed from US-089 contract §4 (names not given there are
// listed in the test-author report):
//
//	type BackupSource interface{ BackupDir() string }
//	type BackupResult struct{ OK, FellBack bool; Error, EffectiveDir, File string }
//	(*DB).now       func() time.Time          injectable clock
//	(*DB).canWrite  func(dir string) error    injectable writability probe
//	(*DB).runBackup(src BackupSource) BackupResult
//	(*DB).StartBackupScheduler(src BackupSource)
//	(*DB).verifyBackup(path string) error     compares against the live database
//	(*DB).verifyCopyCounts(path string, before map[string]int) (retry bool, err error)   (T-6b seam)

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// bkSource is a BackupSource with a fixed folder and an optional delay.
type bkSource struct {
	dir   string
	delay time.Duration
}

func (f bkSource) BackupDir() string {
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	return f.dir
}

func bkDay(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 21, 5, 0, 0, time.Local)
}

// openBackupDB opens <dir>/data.db with a fixed clock.
func openBackupDB(t *testing.T, dir string, now time.Time) *DB {
	t.Helper()
	d, err := New(filepath.Join(dir, "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	d.now = func() time.Time { return now }
	return d
}

// touchBackups creates "<base>.backup-<date>" files for each yyyymmdd.
func bkTouch(t *testing.T, dir, base string, dates ...string) {
	t.Helper()
	for _, dt := range dates {
		if err := os.WriteFile(filepath.Join(dir, base+".backup-"+dt), []byte("old "+dt), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// dateRange returns yyyymmdd strings from start to end inclusive.
func bkDateRange(start, end time.Time) []string {
	var out []string
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		out = append(out, d.Format("20060102"))
	}
	return out
}

func bkNames(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

// backupNames lists strict "data.db.backup-YYYYMMDD" files only.
func bkBackupNames(t *testing.T, dir string) []string {
	var out []string
	for _, n := range bkNames(t, dir) {
		if len(n) == len("data.db.backup-20261031") && strings.HasPrefix(n, "data.db.backup-") && n != "data.db.backup-manual" {
			out = append(out, n)
		}
	}
	return out
}

func bkExists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

// Backup of a WAL database must contain committed data (kept from before US-089).
func TestBackup_ContainsCommittedData(t *testing.T) {
	dir := t.TempDir()
	d := openBackupDB(t, dir, bkDay(2026, 10, 31))
	if _, err := d.Exec(`INSERT INTO users (username, password_hash, role) VALUES ('bk-user','h','admin')`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	res := d.runBackup(bkSource{})
	if !res.OK {
		t.Fatalf("backup failed: %+v", res)
	}
	dest := filepath.Join(dir, "data.db.backup-20261031")
	b, err := sql.Open("sqlite", dest)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	var n int
	if err := b.QueryRow(`SELECT COUNT(*) FROM users WHERE username='bk-user'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("backup missing row: n=%d err=%v", n, err)
	}
}

// T-1 (AC-6): keep the 30 newest.
func TestBackup_T1_KeepsThirtyNewest(t *testing.T) {
	t.Run("31st oldest removed", func(t *testing.T) {
		dir := t.TempDir()
		d := openBackupDB(t, dir, bkDay(2026, 10, 31))
		bkTouch(t, dir, "data.db", bkDateRange(bkDay(2026, 10, 1), bkDay(2026, 10, 30))...)
		if res := d.runBackup(bkSource{}); !res.OK {
			t.Fatalf("backup failed: %+v", res)
		}
		got := bkBackupNames(t, dir)
		if len(got) != 30 {
			t.Fatalf("backups left = %d, want 30: %v", len(got), got)
		}
		if bkExists(dir, "data.db.backup-20261001") {
			t.Error("20261001 should have been removed")
		}
		for _, keep := range []string{"data.db.backup-20261031", "data.db.backup-20261002"} {
			if !bkExists(dir, keep) {
				t.Errorf("%s should remain", keep)
			}
		}
	})
	t.Run("29 files plus run is 30, none removed", func(t *testing.T) {
		dir := t.TempDir()
		d := openBackupDB(t, dir, bkDay(2026, 10, 31))
		bkTouch(t, dir, "data.db", bkDateRange(bkDay(2026, 10, 2), bkDay(2026, 10, 30))...)
		d.runBackup(bkSource{})
		got := bkBackupNames(t, dir)
		if len(got) != 30 || !bkExists(dir, "data.db.backup-20261002") {
			t.Fatalf("backups = %d (%v), want 30 with 20261002 kept", len(got), got)
		}
	})
	t.Run("same-day re-run overwrites and keeps 30", func(t *testing.T) {
		dir := t.TempDir()
		d := openBackupDB(t, dir, bkDay(2026, 10, 31))
		bkTouch(t, dir, "data.db", bkDateRange(bkDay(2026, 10, 2), bkDay(2026, 10, 31))...)
		if res := d.runBackup(bkSource{}); !res.OK {
			t.Fatalf("re-run failed: %+v", res)
		}
		got := bkBackupNames(t, dir)
		if len(got) != 30 || !bkExists(dir, "data.db.backup-20261002") {
			t.Fatalf("backups = %d (%v), want 30 with 20261002 kept", len(got), got)
		}
		info, _ := os.Stat(filepath.Join(dir, "data.db.backup-20261031"))
		if info == nil || info.Size() < 4096 {
			t.Errorf("today's file was not overwritten by a real copy: %v", info)
		}
	})
}

// T-2 (AC-6, AC-7): strict names only.
func TestBackup_T2_PruneIgnoresForeignFiles(t *testing.T) {
	dir := t.TempDir()
	d := openBackupDB(t, dir, bkDay(2026, 11, 2))
	bkTouch(t, dir, "data.db", bkDateRange(bkDay(2026, 10, 1), bkDay(2026, 11, 1))...) // 32 files
	extras := []string{
		"data.db.pre-reset-20261110-143005",
		"data.db.backup-manual",
		"data.db.backup-20261001.tmp",
		"other.db.backup-20250101",
	}
	for _, e := range extras {
		if err := os.WriteFile(filepath.Join(dir, e), []byte("keep"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if res := d.runBackup(bkSource{}); !res.OK {
		t.Fatalf("backup failed: %+v", res)
	}
	got := bkBackupNames(t, dir)
	if len(got) != 30 {
		t.Fatalf("backups left = %d, want 30: %v", len(got), got)
	}
	for _, gone := range []string{"20261001", "20261002", "20261003"} {
		if bkExists(dir, "data.db.backup-"+gone) {
			t.Errorf("%s should have been removed", gone)
		}
	}
	if !bkExists(dir, "data.db.backup-20261004") || !bkExists(dir, "data.db.backup-20261102") {
		t.Error("20261004 and 20261102 must remain")
	}
	for _, e := range extras {
		if !bkExists(dir, e) {
			t.Errorf("%s must never be deleted", e)
		}
	}
}

// T-3 (AC-8): glob metacharacters in the folder name.
func TestBackup_T3_PruneWorksInFolderWithBrackets(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "a[1]")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	d := openBackupDB(t, dir, bkDay(2026, 11, 2))
	bkTouch(t, dir, "data.db", bkDateRange(bkDay(2026, 10, 1), bkDay(2026, 11, 1))...) // 32 files
	if res := d.runBackup(bkSource{}); !res.OK {
		t.Fatalf("backup failed: %+v", res)
	}
	got := bkBackupNames(t, dir)
	if len(got) != 30 {
		t.Fatalf("backups left = %d, want 30: %v", len(got), got)
	}
	for _, gone := range []string{"20261001", "20261002", "20261003"} {
		if bkExists(dir, "data.db.backup-"+gone) {
			t.Errorf("%s should have been removed", gone)
		}
	}
}

// T-4 (AC-10): the copy opens, passes integrity_check and matches live counts.
func TestBackup_T4_CopyVerifiedAndNoTmpLeft(t *testing.T) {
	dir := t.TempDir()
	d := openBackupDB(t, dir, bkDay(2026, 10, 31))
	if _, err := d.Exec(`INSERT INTO users (username, password_hash, role) VALUES ('bk-user','h','admin')`); err != nil {
		t.Fatal(err)
	}
	if res := d.runBackup(bkSource{}); !res.OK {
		t.Fatalf("backup failed: %+v", res)
	}
	dest := filepath.Join(dir, "data.db.backup-20261031")
	b, err := sql.Open("sqlite", dest)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	var ic string
	if err := b.QueryRow(`PRAGMA integrity_check`).Scan(&ic); err != nil || ic != "ok" {
		t.Fatalf("integrity_check = %q err=%v, want ok", ic, err)
	}
	var live, copyN int
	d.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&live)
	if err := b.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&copyN); err != nil || copyN != live || live != 2 {
		t.Errorf("users live=%d copy=%d err=%v, want 2 and 2 (seeded admin + bk-user)", live, copyN, err)
	}
	for _, n := range bkNames(t, dir) {
		if strings.HasSuffix(n, ".tmp") {
			t.Errorf("leftover temp file %s", n)
		}
	}
}

// T-5 (AC-9): a failed copy keeps the day's good file and does not prune.
func TestBackup_T5_FailedCopyKeepsGoodFileAndDoesNotPrune(t *testing.T) {
	dir := t.TempDir()
	d := openBackupDB(t, dir, bkDay(2026, 10, 31))
	older := bkDateRange(bkDay(2026, 9, 30).AddDate(0, 0, -30), bkDay(2026, 9, 30)) // 31 files
	if len(older) != 31 {
		t.Fatalf("fixture has %d older files, want 31", len(older))
	}
	bkTouch(t, dir, "data.db", older...)
	marker := []byte("GOOD-BACKUP-MARKER-BYTES")
	good := filepath.Join(dir, "data.db.backup-20261031")
	if err := os.WriteFile(good, marker, 0o644); err != nil {
		t.Fatal(err)
	}
	// A directory where the temp file must go makes the copy fail.
	if err := os.Mkdir(filepath.Join(dir, "data.db.backup-20261031.tmp"), 0o755); err != nil {
		t.Fatal(err)
	}

	res := d.runBackup(bkSource{})
	if res.OK {
		t.Fatalf("backup reported OK, want failure: %+v", res)
	}
	if res.Error == "" {
		t.Error("Error is empty, want the failure reason")
	}
	got, err := os.ReadFile(good)
	if err != nil || !bytes.Equal(got, marker) {
		t.Errorf("same-day file changed or missing: %q err=%v", got, err)
	}
	for _, dt := range older {
		if !bkExists(dir, "data.db.backup-"+dt) {
			t.Errorf("older file %s was pruned after a failed backup", dt)
		}
	}
}

// T-6 (AC-10): verifyBackup.
func TestBackup_T6_VerifyBackup(t *testing.T) {
	dir := t.TempDir()
	d := openBackupDB(t, dir, bkDay(2026, 10, 31))
	if _, err := d.Exec(`INSERT INTO users (username, password_hash, role) VALUES ('u1','h','admin')`); err != nil {
		t.Fatal(err)
	}

	garbage := filepath.Join(dir, "garbage.bin")
	if err := os.WriteFile(garbage, bytes.Repeat([]byte{0x41}, 4096), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := d.verifyBackup(garbage); err == nil {
		t.Error("verifyBackup(4096 x 0x41) = nil, want error")
	}

	if _, err := d.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(dir, "good-copy.db")
	if _, err := d.Exec(`VACUUM INTO ?`, good); err != nil {
		t.Fatal(err)
	}
	if err := d.verifyBackup(good); err != nil {
		t.Errorf("verifyBackup(good copy) = %v, want nil", err)
	}

	extra := filepath.Join(dir, "extra-row.db")
	if _, err := d.Exec(`VACUUM INTO ?`, extra); err != nil {
		t.Fatal(err)
	}
	c, err := sql.Open("sqlite", extra)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Exec(`INSERT INTO users (username, password_hash, role) VALUES ('ghost','h','admin')`); err != nil {
		t.Fatal(err)
	}
	c.Close()
	if err := d.verifyBackup(extra); err == nil {
		t.Error("verifyBackup(copy with one extra row) = nil, want error")
	}
}

// T-7 (AC-11, AC-12): destination folder and fallback.
func TestBackup_T7_DestinationAndFallback(t *testing.T) {
	today := bkDay(2026, 10, 31)
	setup := func(t *testing.T) (dbDir, cfgDir string, d *DB) {
		root := t.TempDir()
		dbDir = filepath.Join(root, "app")
		cfgDir = filepath.Join(root, "copias")
		for _, p := range []string{dbDir, cfgDir} {
			if err := os.Mkdir(p, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		d = openBackupDB(t, dbDir, today)
		bkTouch(t, cfgDir, "data.db", bkDateRange(bkDay(2026, 10, 1), bkDay(2026, 10, 30))...) // 30 old files
		return
	}
	configuredUntouched := func(t *testing.T, cfgDir string) {
		t.Helper()
		if got := bkBackupNames(t, cfgDir); len(got) != 30 || !bkExists(cfgDir, "data.db.backup-20261001") {
			t.Errorf("configured folder changed: %d files %v, want 30 untouched", len(got), got)
		}
	}
	const dest = "data.db.backup-20261031"

	t.Run("a valid folder gets the file and the db folder gets none", func(t *testing.T) {
		dbDir, cfgDir, d := setup(t)
		res := d.runBackup(bkSource{dir: cfgDir})
		if !res.OK || res.FellBack {
			t.Fatalf("result = %+v, want OK and not fell back", res)
		}
		if res.EffectiveDir != cfgDir {
			t.Errorf("EffectiveDir = %q, want %q", res.EffectiveDir, cfgDir)
		}
		if !bkExists(cfgDir, dest) {
			t.Error("backup missing in configured folder")
		}
		if bkExists(dbDir, dest) {
			t.Error("backup also written next to the db")
		}
		if got := bkBackupNames(t, cfgDir); len(got) != 30 || bkExists(cfgDir, "data.db.backup-20261001") {
			t.Errorf("configured folder not pruned to 30 (oldest gone): %d %v", len(got), got)
		}
	})

	fellBack := func(t *testing.T, name string, prep func(t *testing.T, root, cfgDir string, d *DB) string) {
		t.Run(name, func(t *testing.T) {
			dbDir, cfgDir, d := setup(t)
			bad := prep(t, filepath.Dir(dbDir), cfgDir, d)
			res := d.runBackup(bkSource{dir: bad})
			if !res.OK || !res.FellBack {
				t.Fatalf("result = %+v, want OK and FellBack", res)
			}
			if !strings.Contains(res.Error, bad) {
				t.Errorf("Error %q does not name the failing folder %q", res.Error, bad)
			}
			if res.EffectiveDir != dbDir {
				t.Errorf("EffectiveDir = %q, want default %q", res.EffectiveDir, dbDir)
			}
			if !bkExists(dbDir, dest) {
				t.Error("fallback backup missing next to the db")
			}
			configuredUntouched(t, cfgDir)
		})
	}
	fellBack(t, "missing folder", func(t *testing.T, root, cfgDir string, d *DB) string {
		return filepath.Join(root, "nope")
	})
	fellBack(t, "path is a file", func(t *testing.T, root, cfgDir string, d *DB) string {
		p := filepath.Join(root, "iamafile")
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	})
	fellBack(t, "folder not writable", func(t *testing.T, root, cfgDir string, d *DB) string {
		d.canWrite = func(dir string) error {
			if dir == cfgDir {
				return errors.New("read-only")
			}
			return nil
		}
		return cfgDir
	})

	t.Run("empty dir is the default and not a fallback", func(t *testing.T) {
		dbDir, cfgDir, d := setup(t)
		res := d.runBackup(bkSource{dir: ""})
		if !res.OK || res.FellBack || res.EffectiveDir != dbDir {
			t.Fatalf("result = %+v, want OK, no fallback, dir %q", res, dbDir)
		}
		if !bkExists(dbDir, dest) {
			t.Error("backup missing next to the db")
		}
		configuredUntouched(t, cfgDir)
	})
}

// T-8 (AC-5): the scheduler never waits for the folder source or the backup.
func TestBackup_T8_SchedulerReturnsImmediately(t *testing.T) {
	d := openBackupDB(t, t.TempDir(), bkDay(2026, 10, 31))
	start := time.Now()
	d.StartBackupScheduler(bkSource{delay: 2 * time.Second})
	if el := time.Since(start); el > 200*time.Millisecond {
		t.Errorf("StartBackupScheduler took %v, want under 200ms", el)
	}
	time.Sleep(2500 * time.Millisecond) // let the goroutine finish before the temp dir is removed
}

// T-9 (AC-17): consistent copies while sales are being committed.
func TestBackup_T9_ConsistentWhileSellingWithOpenRegister(t *testing.T) {
	dir := t.TempDir()
	d := openBackupDB(t, dir, bkDay(2026, 10, 27))
	for _, q := range []string{
		`INSERT INTO registers (id, user_id, opening_amount, status) VALUES (1, 1, 50000, 'open')`,
	} {
		if _, err := d.Exec(q); err != nil {
			t.Fatal(err)
		}
	}

	var stop atomic.Bool
	var writes, writerErrs atomic.Int64
	done := make(chan struct{})
	go func() {
		defer close(done)
		for !stop.Load() {
			tx, err := d.Begin()
			if err != nil {
				writerErrs.Add(1)
				continue
			}
			res, err := tx.Exec(`INSERT INTO sales (user_id, register_id, subtotal, total, payment_method, status) VALUES (1, 1, 10000, 10000, 'cash', 'completed')`)
			if err == nil {
				id, _ := res.LastInsertId()
				if _, err = tx.Exec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (?, 'cash', 10000)`, id); err == nil {
					_, err = tx.Exec(`INSERT INTO register_movements (register_id, type, amount, description) VALUES (1, 'sale', 10000, 'sale')`)
				}
			}
			if err != nil {
				tx.Rollback()
				writerErrs.Add(1)
				continue
			}
			if err := tx.Commit(); err != nil {
				writerErrs.Add(1)
				continue
			}
			writes.Add(1)
		}
	}()

	count := func(db *sql.DB, q string) int {
		var n int
		if err := db.QueryRow(q).Scan(&n); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		return n
	}
	for i := 0; i < 5; i++ {
		now := bkDay(2026, 10, 27+i)
		d.now = func() time.Time { return now }
		before := count(d.DB, `SELECT COUNT(*) FROM sales`)
		res := d.runBackup(bkSource{})
		after := count(d.DB, `SELECT COUNT(*) FROM sales`)
		if !res.OK {
			t.Fatalf("backup %d failed: %+v", i, res)
		}
		c, err := sql.Open("sqlite", filepath.Join(dir, "data.db.backup-"+now.Format("20060102")))
		if err != nil {
			t.Fatal(err)
		}
		var ic string
		if err := c.QueryRow(`PRAGMA integrity_check`).Scan(&ic); err != nil || ic != "ok" {
			t.Errorf("backup %d integrity_check = %q err=%v", i, ic, err)
		}
		rows, err := c.Query(`PRAGMA foreign_key_check`)
		if err != nil {
			t.Fatal(err)
		}
		if rows.Next() {
			t.Errorf("backup %d foreign_key_check reported violations", i)
		}
		rows.Close()
		if n := count(c, `SELECT COUNT(*) FROM registers WHERE id = 1 AND status = 'open'`); n != 1 {
			t.Errorf("backup %d: open register rows = %d, want 1", i, n)
		}
		sales := count(c, `SELECT COUNT(*) FROM sales`)
		if sales < before || sales > after {
			t.Errorf("backup %d: sales in copy = %d, want between %d and %d", i, sales, before, after)
		}
		if pay := count(c, `SELECT COUNT(*) FROM sale_payments`); pay != sales {
			t.Errorf("backup %d: sale_payments = %d, sales = %d: half-written sale", i, pay, sales)
		}
		c.Close()
	}
	stop.Store(true)
	<-done
	if n := writerErrs.Load(); n != 0 {
		t.Errorf("writer errors = %d, want 0", n)
	}
	if writes.Load() == 0 {
		t.Error("writer committed 0 sales, the test proved nothing")
	}
}

// bkLiveCounts returns the row count of every user table of d.
func bkLiveCounts(t *testing.T, d *DB) map[string]int {
	t.Helper()
	rows, err := d.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		tables = append(tables, n)
	}
	rows.Close()
	out := map[string]int{}
	for _, tb := range tables {
		var n int
		if err := d.QueryRow(`SELECT COUNT(*) FROM "` + tb + `"`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		out[tb] = n
	}
	return out
}

// T-6b (AC-10, AC-17): row-count tolerance rule, deterministic.
//
// Seam assumed: (*DB).verifyCopyCounts(path string, before map[string]int) (retry bool, err error).
// It reads the live "after" counts itself, checks integrity_check, and requires for every user
// table min(before, after) <= copy <= max(before, after). retry is true only when a table failed
// while its live count changed (before != after).
func TestBackup_T6b_RowCountTolerance(t *testing.T) {
	dir := t.TempDir()
	d := openBackupDB(t, dir, bkDay(2026, 10, 31))
	for i := 101; i <= 103; i++ { // ids clear of the seeded admin (id 1): 4 users live
		if _, err := d.Exec(`INSERT INTO users (id, username, password_hash, role) VALUES (?, ?, 'h', 'cashier')`, i, fmt.Sprintf("u%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 12; i++ { // live "after" = 12 sales of $10,000
		if _, err := d.Exec(`INSERT INTO sales (user_id, subtotal, total, payment_method, status) VALUES (1, 10000, 10000, 'cash', 'completed')`); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := d.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatal(err)
	}
	before := bkLiveCounts(t, d)
	before["sales"] = 10 // 10 before, 12 after

	// mkCopy copies the live db, then sets the copy's sales to salesN and adds extraUsers users.
	mkCopy := func(name string, salesN, extraUsers int) string {
		p := filepath.Join(dir, name)
		if _, err := d.Exec(`VACUUM INTO ?`, p); err != nil {
			t.Fatal(err)
		}
		c, err := sql.Open("sqlite", p)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if salesN < 12 {
			if _, err := c.Exec(`DELETE FROM sales WHERE id > ?`, salesN); err != nil {
				t.Fatal(err)
			}
		}
		for i := 12; i < salesN; i++ {
			if _, err := c.Exec(`INSERT INTO sales (user_id, subtotal, total, payment_method, status) VALUES (1, 10000, 10000, 'cash', 'completed')`); err != nil {
				t.Fatal(err)
			}
		}
		for i := 0; i < extraUsers; i++ {
			if _, err := c.Exec(`INSERT INTO users (username, password_hash, role) VALUES (?, 'h', 'cashier')`, fmt.Sprintf("extra%d", i)); err != nil {
				t.Fatal(err)
			}
		}
		return p
	}

	if retry, err := d.verifyCopyCounts(mkCopy("c11.db", 11, 0), before); err != nil || retry {
		t.Errorf("copy with 11 sales (between 10 and 12): retry=%v err=%v, want nil and no retry", retry, err)
	}
	for _, n := range []int{9, 13} {
		retry, err := d.verifyCopyCounts(mkCopy(fmt.Sprintf("c%d.db", n), n, 0), before)
		if err == nil {
			t.Errorf("copy with %d sales (outside 10..12): err = nil, want failure", n)
		}
		if !retry {
			t.Errorf("copy with %d sales: retry = false, want true (live count changed)", n)
		}
	}
	// users: 4 before and after (seeded admin + 3), copy has 5 -> fails and is NOT retried.
	retry, err := d.verifyCopyCounts(mkCopy("cusers.db", 12, 1), before)
	if err == nil {
		t.Error("copy with 5 users vs 4 live (unchanged): err = nil, want failure")
	}
	if retry {
		t.Error("quiet-table mismatch must not be retried")
	}
}

// T-9b (AC-6): the file date is in the server's local zone, not UTC.
func TestBackup_T9b_FileDateUsesLocalZone(t *testing.T) {
	orig := time.Local
	time.Local = time.FixedZone("UTC-5", -5*3600)
	t.Cleanup(func() { time.Local = orig })

	dir := t.TempDir()
	d := openBackupDB(t, dir, time.Date(2026, 11, 1, 2, 30, 0, 0, time.UTC)) // 31/10/2026 21:30 in UTC-5
	if res := d.runBackup(bkSource{}); !res.OK {
		t.Fatalf("backup failed: %+v", res)
	}
	if !bkExists(dir, "data.db.backup-20261031") {
		t.Errorf("data.db.backup-20261031 missing; dir has %v", bkNames(t, dir))
	}
	if bkExists(dir, "data.db.backup-20261101") {
		t.Error("file dated in UTC (20261101) was written")
	}
}
