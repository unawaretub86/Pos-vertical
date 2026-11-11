package database

import "testing"

// T-15: SchemaVersion() is the last dbMigrations version (23 today) and equals
// MAX(version) of a freshly migrated database.
func TestSchemaVersion_T15_MatchesLastMigrationAndFreshDB(t *testing.T) {
	last := dbMigrations[len(dbMigrations)-1].version
	if got := SchemaVersion(); got != last {
		t.Fatalf("SchemaVersion() = %d, want last dbMigrations version %d", got, last)
	}
	if got := SchemaVersion(); got != 23 {
		t.Fatalf("SchemaVersion() = %d, want 23 today", got)
	}

	d, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	var max int
	if err := d.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&max); err != nil {
		t.Fatal(err)
	}
	if max != SchemaVersion() {
		t.Fatalf("MAX(schema_migrations.version) = %d, SchemaVersion() = %d", max, SchemaVersion())
	}
}
