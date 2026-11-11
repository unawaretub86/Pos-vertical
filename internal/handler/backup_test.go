package handler

// US-089 contract §4, T-12..T-14 (backup settings endpoints).
//
// API assumed (the contract names the endpoints, not the Go symbols):
//
//	GET  /backup/status   admin only -> {"data": {configured_dir, default_dir, effective_dir,
//	                                      last_attempt_at, last_success_at, last_ok, fell_back,
//	                                      last_error, last_file}}
//	PUT  /backup/config   admin only, body {"dir": "..."}; errors are {"error": "<Spanish>"}
//	Handler.canWrite      func(dir string) error, injectable writability probe
//	(*database.DB).RunBackup(src database.BackupSource) database.BackupResult
//	*repository.SetupRepository implements database.BackupSource (BackupDir() string) and
//	records the result of a run into business_config.

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/andreyo/pos/internal/database"
)

type bkStatus struct {
	ConfiguredDir string  `json:"configured_dir"`
	DefaultDir    string  `json:"default_dir"`
	EffectiveDir  string  `json:"effective_dir"`
	LastAttemptAt *string `json:"last_attempt_at"`
	LastSuccessAt *string `json:"last_success_at"`
	LastOK        bool    `json:"last_ok"`
	FellBack      bool    `json:"fell_back"`
	LastError     string  `json:"last_error"`
	LastFile      string  `json:"last_file"`
}

func bkGetStatus(t *testing.T, h *Handler) bkStatus {
	t.Helper()
	w := doAuth(h, tokenFor(t, 1, "admin"), http.MethodGet, "/backup/status", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /backup/status = %d, body %s", w.Code, w.Body.String())
	}
	var env struct {
		Data bkStatus `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v\n%s", err, w.Body.String())
	}
	return env.Data
}

func bkPutDir(h *Handler, token, dir string) (int, string) {
	body, _ := json.Marshal(map[string]string{"dir": dir})
	w := doAuth(h, token, http.MethodPut, "/backup/config", string(body))
	return w.Code, w.Body.String()
}

// bkHandler returns a handler over a file-backed database with the config row
// already created (as the onboarding wizard does), plus the db path.
func bkHandler(t *testing.T) (*Handler, *database.DB, string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "data.db")
	db, err := database.New(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	h := New(db)
	putSetup(t, h, `{"name":"Tienda Uno"}`)
	return h, db, p
}

// T-12 (AC-13)
func TestBackupConfig_T12_PutValidation(t *testing.T) {
	h, _, _ := bkHandler(t)
	admin := tokenFor(t, 1, "admin")
	good := t.TempDir()

	if code, body := bkPutDir(h, admin, good); code != http.StatusOK {
		t.Fatalf("PUT existing dir = %d %s, want 200", code, body)
	}
	if got := bkGetStatus(t, h).ConfiguredDir; got != good {
		t.Errorf("configured_dir = %q, want %q", got, good)
	}

	// Rejections leave the stored value untouched.
	nope := filepath.Join(good, "nope")
	file := filepath.Join(good, "afile.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, dir, msg string
	}{
		{"missing folder", nope, "La carpeta no existe"},
		{"relative path", "Respaldos", "Escriba la ruta completa de la carpeta"},
		{"path is a file", file, ""},
	}
	for _, c := range cases {
		code, body := bkPutDir(h, admin, c.dir)
		if code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400 (%s)", c.name, code, body)
		}
		if c.msg != "" && !strings.Contains(body, c.msg) {
			t.Errorf("%s: body %s does not contain %q", c.name, body, c.msg)
		}
		if got := bkGetStatus(t, h).ConfiguredDir; got != good {
			t.Errorf("%s: configured_dir changed to %q, want %q", c.name, got, good)
		}
	}

	// A missing, null or non-string "dir" is rejected and never clears the value.
	for _, body := range []string{`{}`, `{"dir":null}`, `{"dir":5}`} {
		w := doAuth(h, admin, http.MethodPut, "/backup/config", body)
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "Falta la carpeta") {
			t.Errorf("PUT %s = %d %s, want 400 'Falta la carpeta'", body, w.Code, w.Body.String())
		}
		if got := bkGetStatus(t, h).ConfiguredDir; got != good {
			t.Errorf("PUT %s: configured_dir changed to %q, want %q", body, got, good)
		}
	}

	// Surrounding spaces are trimmed before validation and storage.
	padded := t.TempDir()
	if code, body := bkPutDir(h, admin, "  "+padded+"  "); code != http.StatusOK {
		t.Fatalf("PUT padded dir = %d %s, want 200", code, body)
	}
	if got := bkGetStatus(t, h).ConfiguredDir; got != padded {
		t.Errorf("configured_dir after padded PUT = %q, want trimmed %q", got, padded)
	}
	if code, body := bkPutDir(h, admin, good); code != http.StatusOK {
		t.Fatalf("PUT good dir = %d %s", code, body)
	}

	// A test file cannot be created in the folder.
	origCanWrite := h.canWrite
	h.canWrite = func(dir string) error { return errors.New("read-only") }
	code, body := bkPutDir(h, admin, t.TempDir())
	if code != http.StatusBadRequest || !strings.Contains(body, "No se puede escribir en la carpeta") {
		t.Errorf("unwritable: %d %s, want 400 'No se puede escribir en la carpeta'", code, body)
	}
	if got := bkGetStatus(t, h).ConfiguredDir; got != good {
		t.Errorf("unwritable: configured_dir changed to %q, want %q", got, good)
	}
	h.canWrite = origCanWrite

	// Empty string clears back to the default.
	if code, body := bkPutDir(h, admin, ""); code != http.StatusOK {
		t.Fatalf("PUT empty = %d %s, want 200", code, body)
	}
	if got := bkGetStatus(t, h).ConfiguredDir; got != "" {
		t.Errorf("configured_dir after clear = %q, want empty", got)
	}

	// Authorization.
	if code, _ := bkPutDir(h, tokenFor(t, 2, "cashier"), good); code != http.StatusForbidden {
		t.Errorf("cashier PUT = %d, want 403", code)
	}
	if code, _ := bkPutDir(h, "", good); code != http.StatusUnauthorized {
		t.Errorf("no token PUT = %d, want 401", code)
	}
}

// T-13 (AC-14)
func TestBackupStatus_T13_ReportsRunsAndSurvivesRestart(t *testing.T) {
	h, db, dbPath := bkHandler(t)
	admin := tokenFor(t, 1, "admin")
	defaultDir := filepath.Dir(dbPath)

	st := bkGetStatus(t, h)
	if st.LastAttemptAt != nil {
		t.Errorf("last_attempt_at before any run = %q, want null", *st.LastAttemptAt)
	}

	// Run into a configured, usable folder.
	folder := t.TempDir()
	if code, body := bkPutDir(h, admin, folder); code != http.StatusOK {
		t.Fatalf("PUT dir = %d %s", code, body)
	}
	res := db.RunBackup(h.setup)
	if !res.OK || res.FellBack {
		t.Fatalf("backup into %s: %+v", folder, res)
	}
	st = bkGetStatus(t, h)
	wantSuffix := ".backup-" + time.Now().Format("20060102")
	if !st.LastOK || st.FellBack || st.LastAttemptAt == nil || st.LastSuccessAt == nil {
		t.Errorf("after good run: %+v, want last_ok true, fell_back false, timestamps set", st)
	}
	if filepath.Clean(st.EffectiveDir) != filepath.Clean(folder) {
		t.Errorf("effective_dir = %q, want %q", st.EffectiveDir, folder)
	}
	if !strings.HasSuffix(st.LastFile, wantSuffix) {
		t.Errorf("last_file = %q, want suffix %q", st.LastFile, wantSuffix)
	}
	if filepath.Clean(st.DefaultDir) != filepath.Clean(defaultDir) {
		t.Errorf("default_dir = %q, want %q", st.DefaultDir, defaultDir)
	}

	// The folder disappears (USB unplugged): the run falls back.
	if err := os.RemoveAll(folder); err != nil {
		t.Fatal(err)
	}
	res = db.RunBackup(h.setup)
	if !res.OK || !res.FellBack {
		t.Fatalf("backup into removed folder: %+v, want OK and FellBack", res)
	}
	st = bkGetStatus(t, h)
	if !st.FellBack || st.LastError == "" {
		t.Errorf("after fallback: %+v, want fell_back true and last_error set", st)
	}
	if filepath.Clean(st.EffectiveDir) != filepath.Clean(defaultDir) {
		t.Errorf("effective_dir after fallback = %q, want default %q", st.EffectiveDir, defaultDir)
	}

	// Values survive reopening the database.
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db2, err := database.New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db2.Close() })
	h2 := New(db2)
	st2 := bkGetStatus(t, h2)
	if !st2.FellBack || st2.LastError != st.LastError || st2.ConfiguredDir != st.ConfiguredDir || st2.LastAttemptAt == nil {
		t.Errorf("after reopen: %+v, want the same values as %+v", st2, st)
	}

	if w := doAuth(h2, tokenFor(t, 2, "cashier"), http.MethodGet, "/backup/status", ""); w.Code != http.StatusForbidden {
		t.Errorf("cashier GET status = %d, want 403", w.Code)
	}
	if w := doAuth(h2, "", http.MethodGet, "/backup/status", ""); w.Code != http.StatusUnauthorized {
		t.Errorf("no token GET status = %d, want 401", w.Code)
	}
}

// T-14 (AC-15): the backup settings live apart from /setup.
func TestBackupConfig_T14_IndependentFromSetup(t *testing.T) {
	h, _, _ := bkHandler(t)
	admin := tokenFor(t, 1, "admin")
	dirX := t.TempDir()

	if code, body := bkPutDir(h, admin, dirX); code != http.StatusOK {
		t.Fatalf("PUT dir = %d %s", code, body)
	}
	putSetup(t, h, `{"name":"Tienda Uno"}`)
	if got := bkGetStatus(t, h).ConfiguredDir; got != dirX {
		t.Errorf("configured_dir after PUT /setup = %q, want %q", got, dirX)
	}

	// PUT /backup/config leaves name and the LAN flag alone.
	putSetup(t, h, `{"name":"Tienda Uno","lan_access_enabled":true}`)
	if code, body := bkPutDir(h, admin, ""); code != http.StatusOK {
		t.Fatalf("PUT dir = %d %s", code, body)
	}
	cfg, err := h.setup.Get()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "Tienda Uno" || !cfg.LanAccessEnabled {
		t.Errorf("after PUT /backup/config: name=%q lan=%v, want Tienda Uno / true", cfg.Name, cfg.LanAccessEnabled)
	}

	// GET /setup as a cashier exposes no backup key.
	w := doAuth(h, tokenFor(t, 2, "cashier"), http.MethodGet, "/setup", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /setup = %d", w.Code)
	}
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	for k := range env.Data {
		if strings.Contains(strings.ToLower(k), "backup") {
			t.Errorf("GET /setup as cashier has key %q", k)
		}
	}
}

// T-13 (AC-14) extension: initial values, and a failed run keeps the last success.
func TestBackupStatus_T13_InitialValuesAndFailedRunKeepsLastSuccess(t *testing.T) {
	h, db, dbPath := bkHandler(t)
	defaultDir := filepath.Dir(dbPath)

	st := bkGetStatus(t, h)
	if st.LastAttemptAt != nil || st.LastSuccessAt != nil {
		t.Errorf("initial timestamps = %v / %v, want both null", st.LastAttemptAt, st.LastSuccessAt)
	}
	if st.LastOK || st.FellBack {
		t.Errorf("initial last_ok=%v fell_back=%v, want false/false", st.LastOK, st.FellBack)
	}
	if st.LastError != "" || st.LastFile != "" || st.EffectiveDir != "" || st.ConfiguredDir != "" {
		t.Errorf("initial strings not empty: %+v", st)
	}
	if filepath.Clean(st.DefaultDir) != filepath.Clean(defaultDir) {
		t.Errorf("default_dir = %q, want %q", st.DefaultDir, defaultDir)
	}

	// Good run into the default folder.
	if res := db.RunBackup(h.setup); !res.OK {
		t.Fatalf("first backup: %+v", res)
	}
	good := bkGetStatus(t, h)
	if !good.LastOK || good.LastSuccessAt == nil || good.LastAttemptAt == nil || good.LastFile == "" {
		t.Fatalf("after good run: %+v", good)
	}

	// Block the same-day temp file name with a directory so the next run fails.
	tmp := filepath.Join(defaultDir, "data.db.backup-"+time.Now().Format("20060102")+".tmp")
	if err := os.Mkdir(tmp, 0o755); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond) // timestamps have one-second resolution
	if res := db.RunBackup(h.setup); res.OK {
		t.Fatalf("second backup succeeded, want failure: %+v", res)
	}
	bad := bkGetStatus(t, h)
	if bad.LastOK || bad.LastError == "" {
		t.Errorf("after failed run: last_ok=%v last_error=%q, want false and non-empty", bad.LastOK, bad.LastError)
	}
	if bad.LastSuccessAt == nil || *bad.LastSuccessAt != *good.LastSuccessAt {
		t.Errorf("last_success_at = %v, want unchanged %v", bad.LastSuccessAt, *good.LastSuccessAt)
	}
	if bad.LastFile != good.LastFile {
		t.Errorf("last_file = %q, want unchanged %q", bad.LastFile, good.LastFile)
	}
	if bad.LastAttemptAt == nil {
		t.Fatal("last_attempt_at is null after a failed run")
	}
	t1, err1 := time.Parse(time.RFC3339, *good.LastAttemptAt)
	t2, err2 := time.Parse(time.RFC3339, *bad.LastAttemptAt)
	if err1 != nil || err2 != nil || !t2.After(t1) {
		t.Errorf("last_attempt_at not advanced: %q -> %q (errs %v %v)", *good.LastAttemptAt, *bad.LastAttemptAt, err1, err2)
	}
}
