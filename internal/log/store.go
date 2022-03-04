package log

import (
	"bufio"
	"encoding/binary"
	"os"
	"sync"
)

var (
	enc = binary.BigEndian
)

const (
	lenWidth = 8
)

type store struct {
	*os.File

	mu   *sync.Mutex
	buf  *bufio.Writer
	size uint64
}

func newStore(f *os.File) (*store, error) {
	fileInfo, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}

	return &store{
		File: f,

		mu:   &sync.Mutex{},
		buf:  bufio.NewWriter(f),
		size: uint64(fileInfo.Size()),
	}, nil
}

// Append appends the given bytes to the end of the store file. It
// returns the number of bytes written, the position at which the
// bytes were written, and an error if any.
//
// For every record we first write the number of bytes to be written
// and then the actual bytes. This is necessary in order to read the
// record later. (Goto position -> Read number of bytes to read -> read)
func (s *store) Append(p []byte) (n uint64, pos uint64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pos = s.size

	if err := binary.Write(s.buf, enc, uint64(len(p))); err != nil {
		return 0, 0, err
	}

	bytesWritten, err := s.buf.Write(p)
	if err != nil {
		return 0, 0, err
	}

	bytesWritten += lenWidth // account for record len header
	s.size += uint64(bytesWritten)

	return uint64(bytesWritten), pos, nil
}

// Read reads the bytes in the record at the given position.
// Returns the bytes read as a slice of bytes along with an error
// if any.
//
// We first goto the position given and read the number of bytes
// to be read. Next we, read the required number of bytes from
// pos+lenWidth (where lenWidth is the width of the record length
// in the bytes) into a byte slice and return it.
func (s *store) Read(pos uint64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// flush buffer to disk to avoid stale read
	if err := s.buf.Flush(); err != nil {
		return nil, err
	}

	// read record size from record header
	recordSizeBytes := make([]byte, lenWidth)
	if _, err := s.File.ReadAt(
		recordSizeBytes, int64(pos),
	); err != nil {
		return nil, err
	}
	recordSize := enc.Uint64(recordSizeBytes)

	// read bytes in the record
	recordBytes := make([]byte, recordSize)
	if _, err := s.File.ReadAt(
		recordBytes, int64(pos+lenWidth),
	); err != nil {
		return nil, err
	}

	return recordBytes, nil
}

// ReadAt simply forwards the call on the underkying os.File
func (s *store) ReadAt(p []byte, off int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.buf.Flush(); err != nil {
		return 0, err
	}

	return s.File.ReadAt(p, off)
}

// Close closes this store by closing the underlying file instance
func (s *store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.buf.Flush(); err != nil {
		return err
	}

	return s.File.Close()
}
