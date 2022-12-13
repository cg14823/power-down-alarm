package store

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type sqliteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (OutageStore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS outage (start INT NOT NULL, end INT NOT NULL, duration FLOAT NOT NULL);`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS pulse (id INT PRIMARY KEY, time INT NOT NULL);`)
	if err != nil {
		return nil, err
	}

	return &sqliteStore{db: db}, nil
}

func (s *sqliteStore) Pulse() error {
	_, err := s.db.Exec(`INSERT INTO pulse(id, time) VALUES (1, ?) ON CONFLICT (id) DO UPDATE SET time=excluded.time;`,
		time.Now().UTC().Unix())
	return err
}

func (s *sqliteStore) RecordOutage(start, end time.Time) error {
	_, err := s.db.Exec(`INSERT INTO outage(start, end, duration) VALUES(?, ?, ?)`, start.UTC().Unix(), end.UTC().Unix(), end.Sub(start).Seconds())
	return err
}

func (s *sqliteStore) GetLastPulse() (time.Time, error) {
	var unixSeconds int64
	err := s.db.QueryRow(`SELECT (time) FROM pulse WHERE id=1;`).Scan(&unixSeconds)
	if err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, nil
		}

		return time.Time{}, err
	}

	return time.Unix(unixSeconds, 0), nil
}

func (s *sqliteStore) Close() error {
	return s.db.Close()
}
