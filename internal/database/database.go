package database

import (
	"database/sql"
	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
	path string
}

func New(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	d := &DB{DB: db, path: path}
	if err := d.migrate(); err != nil {
		return nil, err
	}
	return d, nil
}
