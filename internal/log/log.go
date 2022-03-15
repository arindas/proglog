package log

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"

	api "github.com/arindas/proglog/api/log_v1"
)

// Represents an append only Log of records.
//
// A log is paritioned into a collection of segments, sorted by the offsets
// of the records they contain. The last segment is the active segment.
//
// Writes goto the last segment till it's capacity is maxed out. Once it's
// capacity is full, we create new segment and set it as the active segment.
//
// Read operations are serviced by a linear search on the segments to find
// the segment which contains the given offset. If the segment is found, we
// simply utilize its Read() operation to read the record.
type Log struct {
	mu sync.RWMutex

	Dir    string
	Config Config

	activeSegment *segment
	segments      []*segment
}

const (
	DefaultMaxStoreBytes uint64 = 1024
	DefaultMaxIndexBytes uint64 = 1024
)

func (l *Log) newSegment(off uint64) error {
	s, err := newSegment(l.Dir, off, l.Config)
	if err != nil {
		return err
	}

	l.segments = append(l.segments, s)
	l.activeSegment = s
	return nil
}

// Sets up all the segments for the log, by using the store files and
// index files in the Log directory.
func (l *Log) setup() error {
	files, err := ioutil.ReadDir(l.Dir)
	if err != nil {
		return err
	}

	baseOffsets := []uint64{}
	for _, file := range files {
		// file name is the baseOffset
		fileExt := path.Ext(file.Name())

		// process only store files, store files need to be present
		// for a segment to be recoverable
		if fileExt != storeExt {
			continue
		}

		offsetStr := strings.TrimSuffix(file.Name(), fileExt)
		baseOffset, _ := strconv.ParseUint(offsetStr, 10, 0)
		baseOffsets = append(baseOffsets, baseOffset)
	}
	sort.Slice(baseOffsets, func(i, j int) bool {
		return baseOffsets[i] < baseOffsets[j]
	})
	for _, baseOffset := range baseOffsets {
		if err = l.newSegment(baseOffset); err != nil {
			return err
		}
	}

	if l.segments == nil {
		err = l.newSegment(l.Config.Segment.InitialOffset)
	}

	return err
}

func NewLog(dir string, c Config) (*Log, error) {
	if c.Segment.MaxStoreBytes == 0 {
		c.Segment.MaxStoreBytes = DefaultMaxStoreBytes
	}
	if c.Segment.MaxIndexBytes == 0 {
		c.Segment.MaxIndexBytes = DefaultMaxIndexBytes
	}

	l := &Log{Dir: dir, Config: c}

	return l, l.setup()
}

// Appends the given record to the active segment. If the active segment is
// is maxed out, it creates a new segment and sets it as the active segment.
// Returns the offset to which the record was written. In case of errors,
// 0 us returned as the offset, along with the error.
func (l *Log) Append(record *api.Record) (uint64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	off, err := l.activeSegment.Append(record)
	if err != nil {
		return 0, err
	}

	if l.activeSegment.IsMaxed() {
		err = l.newSegment(off + 1)
	}

	return off, err
}

// Read looks for the segment in this log containing this offset, and
// returns the invocation of (*segment).Read() on it.
func (l *Log) Read(off uint64) (*api.Record, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var s *segment = nil

	// segments are sorted by offset. Search for segment with "off"
	for _, seg := range l.segments {
		if seg.baseOffset <= off && off < seg.nextOffset {
			s = seg
			break
		}
	}

	if s == nil || s.nextOffset <= off {
		return nil, api.ErrOffsetOutOfRange{Offset: off}
	}

	return s.Read(off)
}

// Closes all segments associated with this Log.
func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, segment := range l.segments {
		if err := segment.Close(); err != nil {
			return err
		}
	}

	return nil
}

// Removes all files associated with this Log after closing it.
func (l *Log) Remove() error {
	if err := l.Close(); err != nil {
		return err
	}

	return os.RemoveAll(l.Dir)
}

// Resets this log by Remove()-ing it and setting it up by
// creating a new empty segment in this log.
func (l *Log) Reset() error {
	if err := l.Remove(); err != nil {
		return err
	}

	return l.setup()
}

func (l *Log) LowestOffset() (uint64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.segments[0].baseOffset, nil
}

func (l *Log) HighestOffset() (uint64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	off := l.segments[len(l.segments)-1].nextOffset
	if off == 0 {
		return 0, nil
	}

	return off - 1, nil
}

// Removes all segments where the highest offset is less than or
// equal to the givne lowest offset.
func (l *Log) Truncate(lowestOffset uint64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	segments := []*segment{}

	for _, s := range l.segments {
		if s.nextOffset <= lowestOffset+1 {
			if err := s.Remove(); err != nil {
				return err
			}
			continue
		}

		segments = append(segments, s)
	}
	l.segments = segments

	return nil
}

type storeReader struct {
	*store
	off int64
}

func (reader *storeReader) Read(p []byte) (int, error) {
	bytesRead, err := reader.ReadAt(p, reader.off)
	reader.off += int64(bytesRead)

	return bytesRead, err
}

// Returns a continuous io.Reader over all segments in this Log.
//
// This utilizes io.MultiReader to concatenate the io.Reader implementations
// of every segment's store in this log.
func (l *Log) Reader() io.Reader {
	l.mu.RLock()
	defer l.mu.RUnlock()

	readers := make([]io.Reader, 0, len(l.segments))
	for _, s := range l.segments {
		readers = append(readers, &storeReader{
			store: s.store, off: 0,
		})
	}

	return io.MultiReader(readers...)
}
