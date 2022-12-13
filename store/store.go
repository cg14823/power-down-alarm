package store

import "time"

// OutageStore is an interface for a place to store the data.
type OutageStore interface {
	RecordOutage(start, end time.Time) error
	GetLastPulse() (time.Time, error)
	Pulse() error
	Close() error
}
