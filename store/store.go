package store

import (
	"encoding/csv"
	"os"
	"strconv"
	"time"
)

// OutageStore is an interface for a place to store the data
type OutageStore interface {
	RecordOutage(start, end time.Time) error
	Close() error
}

type csvStore struct {
	file   *os.File
	writer *csv.Writer
}

func NewCSVOutageStore(filePath string) (OutageStore, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	writer := csv.NewWriter(file)

	return &csvStore{
		file:   file,
		writer: writer,
	}, nil
}

func (s *csvStore) RecordOutage(start, end time.Time) error {
	return s.writer.Write([]string{start.Format(time.RFC3339), end.Format(time.RFC3339), strconv.FormatFloat(end.Sub(start).Seconds(), 'f', -1, 64)})
}

func (s *csvStore) Close() error {
	s.writer.Flush()
	err := s.file.Close()
	s.file, s.writer = nil, nil
	return err
}
