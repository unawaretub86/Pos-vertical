package repository

import "time"

// dbTime formats a range bound the way SQLite's CURRENT_TIMESTAMP stores
// created_at-style columns: zone-less UTC text "YYYY-MM-DD HH:MM:SS".
// Comparisons against those columns are textual, so bounds must use the
// same format (a time.Time bound keeps its offset or a "+00:00" suffix and
// mis-buckets rows).
func dbTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05")
}

// dbTimePtr is dbTime for an optional bound.
func dbTimePtr(t *time.Time) string {
	return dbTime(*t)
}
