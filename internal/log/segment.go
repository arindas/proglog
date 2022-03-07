package log

import (
	"fmt"
	"os"
	"path"

	api "github.com/arindas/proglog/api/log_v1"
	"google.golang.org/protobuf/proto"
)

// Struct segment represents a Log segment with a store file and a index to speed up reads. It is
// the data structure to unify data store and indexes.
type segment struct {
	store                  *store
	index                  *index
	baseOffset, nextOffset uint64
	config                 Config
}

const (
	storeExt = ".store"
	indexExt = ".index"
)

// Creates a new segment from the given base directory, baseOffset and Config.
// Returns a pointer to the segment along with an error if any.
//
// Creates segment files and index files with the format "${baseOffset}.{.store|.index}"
// Also sets up the nextOffset to be written to.
func newSegment(dir string, baseOffset uint64, c Config) (*segment, error) {
	s := &segment{baseOffset: baseOffset, config: c}

	// create the store
	storeFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseOffset, storeExt)),
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0644,
	)
	if err != nil {
		return nil, err
	}
	if s.store, err = newStore(storeFile); err != nil {
		return nil, err
	}

	// create the index
	indexFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseOffset, indexExt)),
		os.O_RDWR|os.O_CREATE,
		0644,
	)
	if err != nil {
		return nil, err
	}
	if s.index, err = newIndex(indexFile, c); err != nil {
		return nil, err
	}

	// obtain the last relative offset of this segment
	if off, _, err := s.index.Read(-1); err != nil {
		// if segment is empty
		s.nextOffset = baseOffset
	} else {
		// absoluteLastOffset = lastSegmentOffset + baseOffset
		// nextOffset = absoluteLastOffset + 1
		s.nextOffset = baseOffset + uint64(off) + 1
	}

	return s, nil
}

// Appends a new record to this segment. Returns the absolute offset of
// the newly written record along with an error if any.
func (s *segment) Append(record *api.Record) (offset uint64, err error) {
	cur := s.nextOffset
	record.Offset = cur

	serializedRecord, err := proto.Marshal(record)
	if err != nil {
		return 0, err
	}

	_, pos, err := s.store.Append(serializedRecord)
	if err != nil {
		return 0, err
	}

	if err = s.index.Write(
		// index offsets are relative to the segment base offset
		uint32(s.nextOffset-uint64(s.baseOffset)),
		pos,
	); err != nil {
		return 0, err
	}

	s.nextOffset++

	return cur, nil
}

// Reads the record at the given absolute offset. Returns the record read
// along with an error if any.
func (s *segment) Read(off uint64) (*api.Record, error) {
	_, pos, err := s.index.Read(int64(off - s.baseOffset))
	if err != nil {
		return nil, err
	}

	serializedRecord, err := s.store.Read(pos)
	if err != nil {
		return nil, err
	}

	record := &api.Record{}
	err = proto.Unmarshal(serializedRecord, record)

	return record, err
}

// Returns whether this segment capacity is full or not.
func (s *segment) IsMaxed() bool {
	return s.store.size >= s.config.Segment.MaxStoreBytes ||
		s.index.size >= s.config.Segment.MaxIndexBytes
}

func (s *segment) Close() error {
	if err := s.store.Close(); err != nil {
		return err
	}

	if err := s.index.Close(); err != nil {
		return err
	}

	return nil
}

// Closes this segment and removes associated files.
func (s *segment) Remove() error {
	if err := s.Close(); err != nil {
		return err
	}

	if err := os.Remove(s.index.Name()); err != nil {
		return err
	}

	if err := os.Remove(s.store.Name()); err != nil {
		return err
	}

	return nil
}
