package server

import (
	"fmt"
	"sync"
)

// Represents a Record in a Log.
type Record struct {
	Value []byte `json:"value"`

	// Offset at which this record
	// is stored in the log
	Offset uint64 `json:"offset"`
}

var ErrOffsetNotFound = fmt.Errorf("offset not found")

// Log is an ordered collection of records.
type Log struct {
	mu      *sync.Mutex
	records []Record
}

func NewLog() *Log {
	return &Log{mu: &sync.Mutex{}}
}

// Appends a new Record to this Log. Returns the offset at which
// the log was written to, along with an error, if any.
func (log *Log) Append(record Record) (uint64, error) {
	log.mu.Lock()
	defer log.mu.Unlock()

	record.Offset = uint64(len(log.records))
	log.records = append(log.records, record)

	return record.Offset, nil
}

// Reads a Record from the given offset from this Log. Returns
// the read Record along with a nil error, or an empty record
// instance, along with ErrOffsetNotFound error (if the offset
// is not found).
func (log *Log) Read(offset uint64) (Record, error) {
	log.mu.Lock()
	defer log.mu.Unlock()

	if offset >= uint64(len(log.records)) {
		return Record{}, ErrOffsetNotFound
	}

	return log.records[offset], nil
}
