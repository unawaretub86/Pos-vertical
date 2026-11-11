package resetdata

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"reflect"
	"strings"
	"testing"
)

// setDocs replaces the electronic documents of fixture F with one document per
// status, attached to sales 1, 2, 3... (sale_id is unique per document).
func setDocs(t *testing.T, db *sql.DB, statuses ...string) {
	t.Helper()
	mustExec(t, db, `DELETE FROM electronic_documents`)
	for i, s := range statuses {
		mustExec(t, db, `INSERT INTO electronic_documents (id, sale_id, status) VALUES (?, ?, ?)`, "ed-"+s, i+1, s)
	}
}

// T-8: port probe.
func TestGuard_T8_PortProbe(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port

	if !portInUse(port) {
		t.Errorf("portInUse(%d) = false while a listener is open", port)
	}
	err = CheckPort(port)
	if !errors.Is(err, ErrPOSRunning) || !errors.Is(err, ErrRefused) {
		t.Errorf("CheckPort err = %v, want ErrPOSRunning (refused)", err)
	}
	if ExitCode(err) != 1 {
		t.Errorf("ExitCode = %d, want 1", ExitCode(err))
	}

	l.Close()
	if portInUse(port) {
		t.Errorf("portInUse(%d) = true after the listener closed", port)
	}
	if err := CheckPort(port); err != nil {
		t.Errorf("CheckPort on a closed port = %v, want nil", err)
	}
}

// T-8: another connection holding the write lock makes BEGIN EXCLUSIVE busy -> refusal.
func TestGuard_T8_LockBusyRefuses(t *testing.T) {
	_, path := newFixtureF(t)
	db := openRaw(t, path, 100) // short busy timeout: the holder never lets go
	ctx := context.Background()
	before := allCounts(t, db)

	holder := openRaw(t, path, 100)
	conn, err := holder.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.ExecContext(ctx, `ROLLBACK`) //nolint:errcheck
		conn.Close()
	})

	_, err = Apply(ctx, db, Options{})
	if !errors.Is(err, ErrLocked) || !errors.Is(err, ErrRefused) {
		t.Fatalf("Apply err = %v, want ErrLocked (refused)", err)
	}
	if ExitCode(err) != 1 {
		t.Errorf("ExitCode = %d, want 1", ExitCode(err))
	}
	if after := allCounts(t, db); !reflect.DeepEqual(before, after) {
		t.Errorf("counts changed under a busy lock:\nbefore %v\nafter  %v", before, after)
	}
	if got := stockOf(t, db, 1); got != 106 {
		t.Errorf("P1 stock = %v, want 106", got)
	}
}

// T-9: an accepted document refuses the reset whatever fiscal_config.ambiente says.
func TestGuard_T9_AcceptedDocumentRefuses(t *testing.T) {
	for _, ambiente := range []string{"test", "production"} {
		t.Run("ambiente_"+ambiente, func(t *testing.T) {
			db, _ := newFixtureF(t)
			ctx := context.Background()
			mustExec(t, db, `UPDATE fiscal_config SET ambiente = ? WHERE id = 1`, ambiente)
			setDocs(t, db, "accepted")
			before := allCounts(t, db)

			for _, opts := range []Options{{}, {Force: true}} {
				err := CheckGuards(ctx, db, opts)
				if !errors.Is(err, ErrFiscal) || !errors.Is(err, ErrRefused) {
					t.Errorf("CheckGuards(%+v) err = %v, want ErrFiscal (refused)", opts, err)
				}
				_, err = Apply(ctx, db, opts)
				if !errors.Is(err, ErrFiscal) {
					t.Errorf("Apply(%+v) err = %v, want ErrFiscal", opts, err)
				}
				if ExitCode(err) != 1 {
					t.Errorf("ExitCode = %d, want 1", ExitCode(err))
				}
				if after := allCounts(t, db); !reflect.DeepEqual(before, after) {
					t.Errorf("rows deleted by a refused Apply(%+v)", opts)
				}
			}
		})
	}
}

// T-9: --discard-fiscal-documents clears the documents and records the discarded counts.
func TestGuard_T9_DiscardFlagClearsAndRecords(t *testing.T) {
	db, _ := newFixtureF(t)
	setDocs(t, db, "accepted")

	if _, err := Apply(context.Background(), db, Options{DiscardFiscal: true}); err != nil {
		t.Fatalf("Apply with DiscardFiscal: %v", err)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM electronic_documents`); n != 0 {
		t.Errorf("electronic_documents = %d, want 0", n)
	}
	var details string
	if err := db.QueryRow(`SELECT details FROM audit_log WHERE action = 'data_reset'`).Scan(&details); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(details, `"accepted":1`) {
		t.Errorf("details %q do not record \"accepted\":1", details)
	}
}

// T-9: rejected and cancelled documents never block.
func TestGuard_T9_RejectedAndCancelledClearWithoutFlag(t *testing.T) {
	db, _ := newFixtureF(t)
	setDocs(t, db, "rejected", "cancelled")

	if err := CheckGuards(context.Background(), db, Options{}); err != nil {
		t.Fatalf("CheckGuards: %v", err)
	}
	if _, err := Apply(context.Background(), db, Options{}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM electronic_documents`); n != 0 {
		t.Errorf("electronic_documents = %d, want 0", n)
	}
}

// T-9b: in-flight documents (pending, draft) refuse as well.
func TestGuard_T9b_InFlightDocumentsRefuse(t *testing.T) {
	cases := []struct {
		name     string
		statuses []string
		wantMsg  string
	}{
		{"pending", []string{"pending"}, "pending 1"},
		{"draft", []string{"draft"}, "draft 1"},
		{"both", []string{"draft", "pending"}, "draft 1, pending 1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, _ := newFixtureF(t)
			ctx := context.Background()
			setDocs(t, db, c.statuses...)
			before := allCounts(t, db)

			_, err := Apply(ctx, db, Options{})
			if !errors.Is(err, ErrFiscal) || !errors.Is(err, ErrRefused) {
				t.Fatalf("Apply err = %v, want ErrFiscal (refused)", err)
			}
			if !strings.Contains(err.Error(), c.wantMsg) {
				t.Errorf("message %q does not list %q", err.Error(), c.wantMsg)
			}
			if after := allCounts(t, db); !reflect.DeepEqual(before, after) {
				t.Errorf("rows deleted by a refused Apply")
			}

			// The override clears them.
			if _, err := Apply(ctx, db, Options{DiscardFiscal: true}); err != nil {
				t.Fatalf("Apply with DiscardFiscal: %v", err)
			}
			if n := count(t, db, `SELECT COUNT(*) FROM electronic_documents`); n != 0 {
				t.Errorf("electronic_documents = %d, want 0", n)
			}
		})
	}
}

// T-16: no active admin refuses before anything is deleted.
func TestGuard_T16_NoActiveAdminRefuses(t *testing.T) {
	db, _ := newFixtureF(t)
	ctx := context.Background()
	mustExec(t, db, `UPDATE users SET active = 0 WHERE id = 1`)
	before := allCounts(t, db)

	err := CheckGuards(ctx, db, Options{})
	if !errors.Is(err, ErrNoAdmin) || !errors.Is(err, ErrRefused) {
		t.Errorf("CheckGuards err = %v, want ErrNoAdmin (refused)", err)
	}
	_, err = Apply(ctx, db, Options{})
	if !errors.Is(err, ErrNoAdmin) {
		t.Fatalf("Apply err = %v, want ErrNoAdmin", err)
	}
	if ExitCode(err) != 1 {
		t.Errorf("ExitCode = %d, want 1", ExitCode(err))
	}
	if after := allCounts(t, db); !reflect.DeepEqual(before, after) {
		t.Errorf("rows changed by a refused Apply")
	}
}

// T-16: the data_reset row uses the lowest-id ACTIVE admin.
func TestGuard_T16_LowestActiveAdminIsUsed(t *testing.T) {
	db, _ := newFixtureF(t)
	mustExec(t, db, `UPDATE users SET active = 0 WHERE id = 1`)
	mustExec(t, db, `INSERT INTO users (id, username, password_hash, role, active) VALUES (3, 'admin2', 'hash-admin2', 'admin', 1)`)

	if _, err := Apply(context.Background(), db, Options{}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	var uid int
	if err := db.QueryRow(`SELECT user_id FROM audit_log WHERE action = 'data_reset'`).Scan(&uid); err != nil {
		t.Fatal(err)
	}
	if uid != 3 {
		t.Errorf("data_reset user_id = %d, want 3", uid)
	}
}
