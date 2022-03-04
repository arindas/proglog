package log

import (
	"io"
	"os"

	"github.com/tysonmote/gommap"
)

var (
	offWidth uint64 = 4
	posWidth uint64 = 8
	entWidth        = offWidth + posWidth
)

// Represents a index containing <segment relative offset, position in store file> entries.
// It is backed by a file on disk, which is mmap()-ed for frequent access.
//
// Real offsets are translated using segment relative offset + segment start offset. The
// segment start offset is stored in the config.
type index struct {
	file *os.File
	mmap gommap.MMap
	size uint64
}

// Creates a new index instance from the given file and the config.
// Returns the index instance along with an error if any. In case of errors a nil
// index is returned.
//
// This function createss a new index instance with the given file, truncates the
// file to Segment.MaxIndexBytes as mentioned inthe config, and then mmaps the file
// into memory. The mmap() operation uses PROT_READ | PROT_WRITE for prot flags,
// thus allowing both reading an writing. It used MAP_SHARED for the map flag,
// meaning changes are shared, i.e changes to the mapped buffer are carried over to.
// the file.
func newIndex(f *os.File, c Config) (*index, error) {
	idx := &index{file: f}

	fileInfo, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}

	idx.size = uint64(fileInfo.Size())

	if err = os.Truncate(f.Name(), int64(c.Segment.MaxIndexBytes)); err != nil {
		return nil, err
	}

	if idx.mmap, err = gommap.Map(
		idx.file.Fd(),
		gommap.PROT_READ|gommap.PROT_WRITE,
		gommap.MAP_SHARED,
	); err != nil {
		return nil, err
	}

	return idx, nil
}

// Closes the given index. Returns an error if any.
//
// Closes the given index instance by sycing the mmap-ed buffer to the file, and
// further syncing the file to the disk. The file is again truncated to the index
// size to remove the trailing empty space. Finally the file is closed.
func (i *index) Close() error {
	if err := i.mmap.Sync(gommap.MS_SYNC); err != nil {
		return err
	}
	if err := i.file.Sync(); err != nil {
		return err
	}

	if err := i.file.Truncate(
		int64(i.size)); err != nil {
		return err
	}

	return i.file.Close()
}

// Reads and returns the segment relative offset and position in store for a record,
// along with an error if any.
//
// The parameter integer is interpreted as the sequence number. In case of negative
// values, it is interpreted from the end.
// (0 indicates first record, -1 indicates the last record)
func (i *index) Read(in int64) (off uint32, pos uint64, err error) {
	if i.size == 0 {
		return 0, 0, io.EOF
	}

	numEntries := i.size / entWidth
	for in < 0 {
		in += int64(numEntries)
	}

	pos = uint64(in) * entWidth

	if i.size < pos+entWidth {
		return 0, 0, io.EOF
	}

	off = enc.Uint32(i.mmap[pos : pos+offWidth])
	pos = enc.Uint64(i.mmap[pos+offWidth : pos+entWidth])

	return off, pos, nil
}

// Writes the given offset and the position pair at the end of the index file.
// Returns an error if any.
//
// The offset and the position are binary encoded and written at the end of
// the file. If the file doesn't have enough space to contain 12 bytes, EOF
// is returned. We also increase the size of the file by 12 bytes, post writing.
func (i *index) Write(off uint32, pos uint64) error {
	if uint64(len(i.mmap)) < i.size+entWidth {
		return io.EOF
	}

	enc.PutUint32(i.mmap[i.size:i.size+offWidth], off)
	enc.PutUint64(i.mmap[i.size+offWidth:i.size+entWidth], pos)

	i.size += entWidth

	return nil
}

func (i *index) Name() string { return i.file.Name() }
