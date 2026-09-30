package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const maxBackups = 7

// StartBackupScheduler runs a backup immediately and then every 24 hours.
// It runs in a background goroutine and never causes the process to exit.
func (d *DB) StartBackupScheduler() {
	go func() {
		d.runBackup()
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			d.runBackup()
		}
	}()
}

func (d *DB) runBackup() {
	date := time.Now().Format("20060102")
	dest := fmt.Sprintf("%s.backup-%s", d.path, date)

	_ = os.Remove(dest) // VACUUM INTO fails if destination exists

	if err := d.checkpoint(); err != nil {
		log.Printf("[backup] checkpoint failed: %v", err)
	}

	_, err := d.Exec("VACUUM INTO ?", dest)
	if err != nil {
		log.Printf("[backup] VACUUM INTO failed: %v", err)
		return
	}

	log.Printf("[backup] created %s", dest)

	if err := pruneBackups(d.path); err != nil {
		log.Printf("[backup] prune failed: %v", err)
	}
}

func (d *DB) checkpoint() error {
	_, err := d.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return err
}



func pruneBackups(dbPath string) error {
	pattern := dbPath + ".backup-*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	if len(matches) <= maxBackups {
		return nil
	}

	sort.Strings(matches) // lexicographic = chronological for YYYYMMDD
	toDelete := matches[:len(matches)-maxBackups]
	for _, f := range toDelete {
		if err := os.Remove(f); err != nil {
			log.Printf("[backup] could not remove old backup %s: %v", f, err)
		} else {
			log.Printf("[backup] removed old backup %s", f)
		}
	}
	return nil
}
