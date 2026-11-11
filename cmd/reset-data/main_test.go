package main

import (
	"bytes"
	"database/sql"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/andreyo/pos/internal/database"
)

// The CLI is a thin wrapper: run(args, stdin, stdout, stderr) returns the exit code.
// Stdin is always injected, never os.Stdin.

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return p
}

// newDB creates data.db with 4 sales and 1 register (default admin id 1 comes from migration 1).
func newDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data.db")
	d, err := database.New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	stmts := []string{
		`INSERT INTO registers (id, user_id, opening_amount, status) VALUES (1, 1, 1000, 'open')`,
		`INSERT INTO sales (id, user_id, register_id, subtotal, total, payment_method) VALUES
			(1, 1, 1, 100, 100, 'cash'), (2, 1, 1, 200, 200, 'cash'), (3, 1, 1, 300, 300, 'cash'), (4, 1, 1, 400, 400, 'cash')`,
	}
	for _, s := range stmts {
		if _, err := d.Exec(s); err != nil {
			t.Fatalf("seed %q: %v", s, err)
		}
	}
	return path
}

func countRows(t *testing.T, path, table string) int {
	t.Helper()
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func exec(t *testing.T, path, q string) {
	t.Helper()
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(q); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}

func backups(t *testing.T, path string) []string {
	t.Helper()
	m, err := filepath.Glob(path + ".pre-reset-*")
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func runCLI(args []string, stdin string) (int, string) {
	var out, errb bytes.Buffer
	code := run(args, strings.NewReader(stdin), &out, &errb)
	return code, out.String() + errb.String()
}

// T-14: a missing database file is exit 2 and no empty file is created.
func TestRun_T14_MissingFileExit2(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")
	code, out := runCLI([]string{"--db", path, "--port", strconv.Itoa(freePort(t))}, "")
	if code != 2 {
		t.Errorf("exit = %d, want 2 (output %q)", code, out)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("tool created %s (stat err %v)", path, err)
	}
}

// T-14: a bad flag is a usage error.
func TestRun_T14_BadFlagExit2(t *testing.T) {
	code, _ := runCLI([]string{"--no-such-flag"}, "")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
}

// T-14 / AC-14: a schema version different from the binary's is a refusal (exit 1), nothing deleted.
func TestRun_T14_SchemaVersionMismatchExit1(t *testing.T) {
	path := newDB(t)
	exec(t, path, `DELETE FROM schema_migrations WHERE version = 23`)
	code, out := runCLI([]string{"--db", path, "--apply", "--confirm", "BORRAR 4", "--port", strconv.Itoa(freePort(t))}, "")
	if code != 1 {
		t.Errorf("exit = %d, want 1 (output %q)", code, out)
	}
	if n := countRows(t, path, "sales"); n != 4 {
		t.Errorf("sales = %d, want 4", n)
	}
	if len(backups(t, path)) != 0 {
		t.Errorf("a backup was created before the schema check")
	}
}

// AC-1 / T-1: dry-run is the default, prints counts, changes nothing, creates no backup.
func TestRun_T1_DryRunDefault(t *testing.T) {
	path := newDB(t)
	code, out := runCLI([]string{"--db", path, "--port", strconv.Itoa(freePort(t))}, "")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (output %q)", code, out)
	}
	if !strings.Contains(out, "sales 4") {
		t.Errorf("output does not contain %q:\n%s", "sales 4", out)
	}
	if n := countRows(t, path, "sales"); n != 4 {
		t.Errorf("sales = %d after dry-run, want 4", n)
	}
	if len(backups(t, path)) != 0 {
		t.Errorf("dry-run created a backup")
	}
}

// T-9b: dry-run with an accepted document exits 0, prints the refusal reason, deletes nothing.
func TestRun_T9b_DryRunWithAcceptedDocExit0(t *testing.T) {
	path := newDB(t)
	exec(t, path, `INSERT INTO electronic_documents (id, sale_id, status) VALUES ('ed-1', 1, 'accepted')`)
	code, out := runCLI([]string{"--db", path, "--port", strconv.Itoa(freePort(t))}, "")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (output %q)", code, out)
	}
	if !strings.Contains(out, "accepted 1") {
		t.Errorf("dry-run did not print the refusal reason with %q:\n%s", "accepted 1", out)
	}
	if n := countRows(t, path, "sales"); n != 4 {
		t.Errorf("sales = %d, want 4", n)
	}
	if n := countRows(t, path, "electronic_documents"); n != 1 {
		t.Errorf("electronic_documents = %d, want 1", n)
	}
}

// T-9: --apply with an accepted document is refused before the backup; --force does not override.
func TestRun_T9_ApplyWithAcceptedDocRefusedNoBackup(t *testing.T) {
	path := newDB(t)
	exec(t, path, `INSERT INTO electronic_documents (id, sale_id, status) VALUES ('ed-1', 1, 'accepted')`)
	port := strconv.Itoa(freePort(t))
	for _, extra := range [][]string{nil, {"--force"}} {
		args := append([]string{"--db", path, "--apply", "--confirm", "BORRAR 4", "--port", port}, extra...)
		code, out := runCLI(args, "")
		if code != 1 {
			t.Errorf("args %v: exit = %d, want 1 (output %q)", extra, code, out)
		}
		if len(backups(t, path)) != 0 {
			t.Errorf("args %v: a backup was created before the fiscal refusal", extra)
		}
		if n := countRows(t, path, "sales"); n != 4 {
			t.Errorf("args %v: sales = %d, want 4", extra, n)
		}
	}
}

// T-16: --apply without an active admin is refused before the backup.
func TestRun_T16_NoActiveAdminRefusedNoBackup(t *testing.T) {
	path := newDB(t)
	exec(t, path, `UPDATE users SET active = 0 WHERE role = 'admin'`)
	code, out := runCLI([]string{"--db", path, "--apply", "--confirm", "BORRAR 4", "--port", strconv.Itoa(freePort(t))}, "")
	if code != 1 {
		t.Errorf("exit = %d, want 1 (output %q)", code, out)
	}
	if len(backups(t, path)) != 0 {
		t.Errorf("a backup file was created")
	}
	if n := countRows(t, path, "sales"); n != 4 {
		t.Errorf("sales = %d, want 4", n)
	}
}

// T-8 / AC-7: --apply while something listens on the POS port is refused before the backup.
func TestRun_T8_PortInUseRefusedNoBackup(t *testing.T) {
	path := newDB(t)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port

	code, out := runCLI([]string{"--db", path, "--apply", "--confirm", "BORRAR 4", "--port", strconv.Itoa(port)}, "")
	if code != 1 {
		t.Errorf("exit = %d, want 1 (output %q)", code, out)
	}
	if len(backups(t, path)) != 0 {
		t.Errorf("a backup file was created")
	}
	if n := countRows(t, path, "sales"); n != 4 {
		t.Errorf("sales = %d, want 4", n)
	}
}

// T-11 / AC-10: a wrong typed confirmation deletes nothing.
func TestRun_T11_WrongConfirmationExit1(t *testing.T) {
	path := newDB(t)
	port := strconv.Itoa(freePort(t))

	cases := []struct {
		name  string
		args  []string
		stdin string
	}{
		{"flag lowercase", []string{"--confirm", "borrar 4"}, ""},
		{"flag wrong number", []string{"--confirm", "BORRAR 5"}, ""},
		{"stdin si", nil, "si\n"},
		{"stdin closed", nil, ""},
	}
	for _, c := range cases {
		args := append([]string{"--db", path, "--apply", "--port", port}, c.args...)
		code, out := runCLI(args, c.stdin)
		if code != 1 {
			t.Errorf("%s: exit = %d, want 1 (output %q)", c.name, code, out)
		}
		if n := countRows(t, path, "sales"); n != 4 {
			t.Errorf("%s: sales = %d, want 4", c.name, n)
		}
	}
}

// AC-2, AC-6, AC-10: --apply with --confirm clears the sales and leaves a backup named *.pre-reset-*.
func TestRun_T2_ApplyWithConfirmFlag(t *testing.T) {
	path := newDB(t)
	dailyBefore, _ := filepath.Glob(path + ".backup-*")
	code, out := runCLI([]string{"--db", path, "--apply", "--confirm", "BORRAR 4", "--port", strconv.Itoa(freePort(t))}, "")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (output %q)", code, out)
	}
	if n := countRows(t, path, "sales"); n != 0 {
		t.Errorf("sales = %d, want 0", n)
	}
	b := backups(t, path)
	if len(b) != 1 {
		t.Fatalf("backups = %v, want exactly 1", b)
	}
	if !strings.Contains(out, filepath.Base(b[0])) {
		t.Errorf("output does not print the backup name %q:\n%s", filepath.Base(b[0]), out)
	}
	if n := countRows(t, b[0], "sales"); n != 4 {
		t.Errorf("backup holds %d sales, want 4", n)
	}
	after, _ := filepath.Glob(path + ".backup-*")
	if !reflect.DeepEqual(dailyBefore, after) {
		t.Errorf(".backup-* files changed by the tool: before %v, after %v", dailyBefore, after)
	}
}

// AC-10: the confirmation can be typed on stdin.
func TestRun_T11_ApplyWithTypedStdin(t *testing.T) {
	path := newDB(t)
	code, out := runCLI([]string{"--db", path, "--apply", "--port", strconv.Itoa(freePort(t))}, "BORRAR 4\n")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (output %q)", code, out)
	}
	if n := countRows(t, path, "sales"); n != 0 {
		t.Errorf("sales = %d, want 0", n)
	}
}

// T-10 / AC-9: a second run is refused unless --force.
func TestRun_T10_SecondRunNeedsForce(t *testing.T) {
	path := newDB(t)
	port := strconv.Itoa(freePort(t))
	if code, out := runCLI([]string{"--db", path, "--apply", "--confirm", "BORRAR 4", "--port", port}, ""); code != 0 {
		t.Fatalf("first run exit = %d (output %q)", code, out)
	}
	exec(t, path, `INSERT INTO sales (user_id, subtotal, total, payment_method) VALUES (1, 100, 100, 'cash')`)

	code, _ := runCLI([]string{"--db", path, "--apply", "--confirm", "BORRAR 1", "--port", port}, "")
	if code != 1 {
		t.Errorf("second run exit = %d, want 1", code)
	}
	if n := countRows(t, path, "sales"); n != 1 {
		t.Errorf("sales = %d after refused second run, want 1", n)
	}
	code, out := runCLI([]string{"--db", path, "--apply", "--force", "--confirm", "BORRAR 1", "--port", port}, "")
	if code != 0 {
		t.Fatalf("forced run exit = %d (output %q)", code, out)
	}
	if n := countRows(t, path, "sales"); n != 0 {
		t.Errorf("sales = %d after forced run, want 0", n)
	}
}
