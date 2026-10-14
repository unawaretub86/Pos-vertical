package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Backup of a WAL database must contain committed data.
func TestBackup_ContainsCommittedData(t *testing.T) {
	p := filepath.Join(t.TempDir(), "b.db")
	d, err := New(p)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if _, err := d.Exec(`INSERT INTO users (username, password_hash, role) VALUES ('bk-user','h','admin')`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	d.runBackup()
	dest := fmt.Sprintf("%s.backup-%s", p, time.Now().Format("20060102"))
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("no backup file: %v", err)
	}
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
