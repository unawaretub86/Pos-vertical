package resetdata

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// T-7: backup is a verified VACUUM INTO copy named <db>.pre-reset-YYYYMMDD-HHMMSS.
func TestBackup_T7_CopyIsVerifiedAndNamedAside(t *testing.T) {
	db, path := newFixtureF(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 10, 14, 30, 5, 0, time.Local)

	want := allCounts(t, db)
	info, err := Backup(ctx, db, path, now)
	if err != nil {
		t.Fatalf("Backup: %v", err)
	}

	wantPath := path + ".pre-reset-20261110-143005"
	if info.Path != wantPath {
		t.Errorf("backup path = %q, want %q", info.Path, wantPath)
	}
	st, err := os.Stat(info.Path)
	if err != nil {
		t.Fatalf("backup file missing: %v", err)
	}
	if st.Size() == 0 || info.Size != st.Size() {
		t.Errorf("backup size reported %d, on disk %d", info.Size, st.Size())
	}
	if m, _ := filepath.Match("data.db.backup-*", filepath.Base(info.Path)); m {
		t.Errorf("backup name %q matches the POS prune glob <db>.backup-*", info.Path)
	}
	if !reflect.DeepEqual(info.Counts, want) {
		t.Errorf("BackupInfo.Counts differ from source:\nwant %v\ngot  %v", want, info.Counts)
	}

	// Open the copy and check it independently.
	cp := openRaw(t, info.Path, 5000)
	var integrity string
	if err := cp.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil {
		t.Fatal(err)
	}
	if integrity != "ok" {
		t.Errorf("integrity_check = %q, want ok", integrity)
	}
	if got := allCounts(t, cp); !reflect.DeepEqual(got, want) {
		t.Errorf("copy counts differ:\nwant %v\ngot  %v", want, got)
	}
	if got := stockOf(t, cp, 1); got != 106 {
		t.Errorf("copy P1 stock = %v, want 106", got)
	}
}

// T-7 (failure): the copy cannot be created -> ErrBackup, exit code 3, nothing deleted.
func TestBackup_T7_FailureMapsToExit3AndDeletesNothing(t *testing.T) {
	db, path := newFixtureF(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 10, 14, 30, 5, 0, time.Local)
	before := allCounts(t, db)

	// A directory already sits at the target name, so VACUUM INTO cannot create the file.
	if err := os.Mkdir(path+".pre-reset-20261110-143005", 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Backup(ctx, db, path, now)
	if err == nil {
		t.Fatal("Backup succeeded although the target cannot be created")
	}
	if !errors.Is(err, ErrBackup) {
		t.Errorf("err = %v, want ErrBackup", err)
	}
	if ExitCode(err) != 3 {
		t.Errorf("ExitCode = %d, want 3", ExitCode(err))
	}
	if after := allCounts(t, db); !reflect.DeepEqual(before, after) {
		t.Errorf("rows changed after backup failure:\nbefore %v\nafter  %v", before, after)
	}
}
