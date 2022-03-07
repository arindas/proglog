package log

import (
	"io/ioutil"
	"os"
	"testing"

	api "github.com/arindas/proglog/api/log_v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func testAppendRead(t *testing.T, emptyLog *Log) {
	t.Helper()
	log := emptyLog

	rec := &api.Record{Value: []byte("hello world")}

	off, err := log.Append(rec)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)

	readRec, err := log.Read(off)
	require.NoError(t, err)
	require.Equal(t, rec.Value, readRec.Value)
}

func testOutOfRangeErr(t *testing.T, emptyLog *Log) {
	t.Helper()
	log := emptyLog

	readRec, err := log.Read(1)
	require.Nil(t, readRec)
	require.Error(t, err)
}

func testInitExisting(t *testing.T, emptyLog *Log) {
	t.Helper()
	oldLog := emptyLog

	recToAppend := &api.Record{Value: []byte("hello world")}

	for i := 0; i < 3; i++ {
		_, err := oldLog.Append(recToAppend)
		require.NoError(t, err)
	}
	require.NoError(t, oldLog.Close())

	off, err := oldLog.LowestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)
	off, err = oldLog.HighestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(2), off)

	newLog, err := NewLog(oldLog.Dir, oldLog.Config)
	require.NoError(t, err)

	off, err = newLog.LowestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)
	off, err = newLog.HighestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(2), off)
}

func testReader(t *testing.T, emptyLog *Log) {
	t.Helper()
	log := emptyLog

	recToAppend := &api.Record{Value: []byte("hello world")}

	off, err := log.Append(recToAppend)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)

	reader := log.Reader()
	bytes, err := ioutil.ReadAll(reader)
	require.NoError(t, err)

	readRecord := &api.Record{}
	err = proto.Unmarshal(bytes[lenWidth:], readRecord)
	require.NoError(t, err)
	require.Equal(t, recToAppend.Value, readRecord.Value)
}

func testTruncate(t *testing.T, emptyLog *Log) {
	t.Helper()
	log := emptyLog

	recToAppend := &api.Record{Value: []byte("hello world")}

	for i := 0; i < 3; i++ {
		_, err := log.Append(recToAppend)
		require.NoError(t, err)
	}

	err := log.Truncate(1)
	require.NoError(t, err)

	_, err = log.Read(0)
	require.Error(t, err)
}

func TestLog(t *testing.T) {
	for scenario, fn := range map[string]func(*testing.T, *Log){
		"append and read a record succeeds": testAppendRead,
		"offset out of range error":         testOutOfRangeErr,
		"init with existing segments":       testInitExisting,
		"reader":                            testReader,
		"truncate":                          testTruncate,
	} {
		t.Run(scenario, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "log-test")
			require.NoError(t, err)
			defer os.RemoveAll(dir)

			c := Config{}
			c.Segment.MaxStoreBytes = 32
			log, err := NewLog(dir, c)
			require.NoError(t, err)

			fn(t, log)
		})

	}
}
