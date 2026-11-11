package resetdata

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// T-14: exit code mapping.
func TestExitCode_T14_Table(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"refused", ErrRefused, 1},
		{"refused wrapped", fmt.Errorf("guard: %w", ErrRefused), 1},
		{"already reset", ErrAlreadyReset, 1},
		{"fiscal", ErrFiscal, 1},
		{"no admin", ErrNoAdmin, 1},
		{"confirmation", ErrConfirmation, 1},
		{"pos running", ErrPOSRunning, 1},
		{"locked", ErrLocked, 1},
		{"schema version", ErrSchemaVersion, 1},
		{"usage", ErrUsage, 2},
		{"usage wrapped", fmt.Errorf("open: %w", ErrUsage), 2},
		{"backup", ErrBackup, 3},
		{"backup wrapped", fmt.Errorf("copy: %w", ErrBackup), 3},
		{"database", ErrDatabase, 4},
		{"database wrapped", fmt.Errorf("tx: %w", ErrDatabase), 4},
	}
	for _, c := range cases {
		if got := ExitCode(c.err); got != c.want {
			t.Errorf("%s: ExitCode(%v) = %d, want %d", c.name, c.err, got, c.want)
		}
	}
}

// T-14: a missing database file is a usage error and no empty file is created.
func TestOpen_T14_MissingFileIsUsageErrorAndCreatesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")
	db, err := Open(path)
	if err == nil {
		db.Close()
		t.Fatal("Open succeeded on a missing file")
	}
	if !errors.Is(err, ErrUsage) || ExitCode(err) != 2 {
		t.Errorf("err = %v (code %d), want ErrUsage / 2", err, ExitCode(err))
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Errorf("Open created %s (stat err %v)", path, statErr)
	}
}

// T-14: schema version differing from the binary's is a refusal.
func TestCheckSchemaVersion_T14(t *testing.T) {
	db, _ := newFixtureF(t)
	ctx := context.Background()

	if err := CheckSchemaVersion(ctx, db); err != nil {
		t.Fatalf("current schema (22) refused: %v", err)
	}

	mustExec(t, db, `DELETE FROM schema_migrations WHERE version = 22`)
	if n := count(t, db, `SELECT MAX(version) FROM schema_migrations`); n != 21 {
		t.Fatalf("setup: max version = %d, want 21", n)
	}
	err := CheckSchemaVersion(ctx, db)
	if !errors.Is(err, ErrSchemaVersion) || !errors.Is(err, ErrRefused) {
		t.Fatalf("err = %v, want ErrSchemaVersion (refused)", err)
	}
	if ExitCode(err) != 1 {
		t.Errorf("ExitCode = %d, want 1", ExitCode(err))
	}
}
