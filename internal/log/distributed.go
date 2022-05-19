package log

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"

	"google.golang.org/protobuf/proto"

	api "github.com/arindas/proglog/api/log_v1"
)

// DistributedLog implements a Raft consensus driven replicated log.
type DistributedLog struct {
	config Config
	log    *Log
	raft   *raft.Raft
}

func NewDistributedLog(dataDir string, config Config) (*DistributedLog, error) {
	l := &DistributedLog{config: config}

	if err := l.setupLog(dataDir); err != nil {
		return nil, err
	}

	if err := l.setupRaft(dataDir); err != nil {
		return nil, err
	}

	return l, nil
}

// Allocates the local log's data structures and creates this log's
// backing files in the given directory.
func (l *DistributedLog) setupLog(dataDir string) error {
	logDir := filepath.Join(dataDir, "log")
	var err error
	if err = os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	l.log, err = NewLog(logDir, l.config)
	return err
}

// Configures replication with the raft consensus protocol.
// A raft instance contains four components:
// - A finite state machine which represents the state of
//  the raft instance. All raft commands are applied to the
//  finite state machine. The FSM then inteprets the command
//  accordingly to produce desired side effects and goto the
//  desired state.
// - A log store for storing the commands to be applied
//  (distinct from our actual record log store)
// - A stable key value store for storing Raft's configuration
//  and cluster data. (e.g addresses of other servers in the
//  cluster)
// - A file snapshot store for storing data snapshots. These
//  are used for data recovery in the event of server failures.
// - A network streaming layer for connecting to other servers
//  in the cluster.
//
// The provided directory path is used for creating all the data
// stores. All persistent data is stored in this directory.
//
// Returns the error ocurred, otherwise nil.
func (l *DistributedLog) setupRaft(dataDir string) error {
	// finite state machine for appying commands to raft
	fsm := &fsm{log: l.log}

	// log store for storing the commands to be applied
	logDir := filepath.Join(dataDir, "raft", "log")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}
	logConfig := l.config
	logConfig.Segment.InitialOffset = 1
	logStore, err := newLogStore(logDir, logConfig)
	if err != nil {
		return err
	}

	// stable store for storing Raft's cluster configuration
	// for instance: the servers in the cluster, their addresses etc.
	stableStore, err := raftboltdb.NewBoltStore(
		filepath.Join(dataDir, "raft", "stable"),
	)
	if err != nil {
		return err
	}

	// snapshot store for storing compact snapshots of Raft's data
	// useful for recovering from failures
	retain := 1
	snapshotStore, err := raft.NewFileSnapshotStore(
		filepath.Join(dataDir, "raft"),
		retain,
		os.Stderr,
	)
	if err != nil {
		return err
	}

	// transport to connect to Raft cluster peers
	maxPool := 5
	timeout := 10 * time.Second
	transport := raft.NewNetworkTransport(
		l.config.Raft.StreamLayer,
		maxPool,
		timeout,
		os.Stderr,
	)

	// override default config with user provided config, if present
	config := raft.DefaultConfig()
	config.LocalID = l.config.Raft.LocalID
	if l.config.Raft.HeartbeatTimeout != 0 {
		config.HeartbeatTimeout = l.config.Raft.HeartbeatTimeout
	}
	if l.config.Raft.ElectionTimeout != 0 {
		config.ElectionTimeout = l.config.Raft.ElectionTimeout
	}
	if l.config.Raft.LeaderLeaseTimeout != 0 {
		config.LeaderLeaseTimeout = l.config.Raft.LeaderLeaseTimeout
	}
	if l.config.Raft.CommitTimeout != 0 {
		config.CommitTimeout = l.config.Raft.CommitTimeout
	}

	l.raft, err = raft.NewRaft(
		config, fsm, logStore, stableStore, snapshotStore, transport,
	)
	if err != nil {
		return err
	}

	hasState, err := raft.HasExistingState(
		logStore, stableStore, snapshotStore,
	)
	if err != nil {
		return err
	}

	// Initially a single server configured with itself as the only
	// voter is setup. Once it becomes a leader, we tell it to add
	// more servers to the cluster. The subsequently added server's
	// don't bootstrap.

	// Used for bootstrapping the Raft cluster with invoking node
	// as the leader.
	if l.config.Raft.BootStrap && !hasState {
		config := raft.Configuration{
			Servers: []raft.Server{
				{ID: config.LocalID, Address: transport.LocalAddr()},
			},
		}
		err = l.raft.BootstrapCluster(config).Error()
	}

	return err
}

func (l *DistributedLog) Append(record *api.Record) (uint64, error) {
	res, err := l.apply(AppendRequestType, &api.ProduceRequest{Record: record})
	if err != nil {
		return 0, err
	}
	return res.(*api.ProduceResponse).Offset, nil
}

// Applies a request to the local Raft instances state machine.
func (l *DistributedLog) apply(reqType RequestType, req proto.Message) (
	interface{}, error,
) {
	var buf bytes.Buffer

	// encode reqType to bytes and write to buffer
	_, err := buf.Write([]byte{byte(reqType)})
	if err != nil {
		return nil, err
	}

	b, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}

	// marhsal the request to bytes and write to buffer
	_, err = buf.Write(b)
	if err != nil {
		return nil, err
	}

	// buffer now contains -> concat(0b{reqType}, 0b{req})

	timeout := 10 * time.Second
	future := l.raft.Apply(buf.Bytes(), timeout)
	if future.Error() != nil {
		return nil, future.Error()
	}

	res := future.Response()
	if err, ok := res.(error); ok {
		return nil, err
	}

	return res, nil
}

func (l *DistributedLog) Read(offset uint64) (*api.Record, error) {
	return l.log.Read(offset)
}

// Serf cluster membership "join" event handler.
func (l *DistributedLog) Join(id, addr string) error {
	configFuture := l.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return err
	}

	serverID, serverAddress := raft.ServerID(id), raft.ServerAddress(addr)

	// remove any existing server with matching serverId or serverAddress
	for _, server := range configFuture.Configuration().Servers {
		if server.ID == serverID || server.Address == serverAddress {
			if server.ID == serverID && server.Address == serverAddress {
				// server has already joined
				return nil
			}

			// remove the existing server
			removeFuture := l.raft.RemoveServer(serverID, 0, 0)
			if err := removeFuture.Error(); err != nil {
				return err
			}
		}
	}

	// add new server as a voter to this raft cluster
	addFuture := l.raft.AddVoter(serverID, serverAddress, 0, 0)
	if err := addFuture.Error(); err != nil {
		return err
	}

	return nil
}

// Serf "leave" event handler for our Raft cluster.
func (l *DistributedLog) Leave(id string) error {
	return l.raft.RemoveServer(raft.ServerID(id), 0, 0).Error()
}

// Waits for leader to be elected synchronously.
// We check every second upto the given timeout duration whether a leader
// has been elected or not. If the leader is elected at some tick second
// we return. Otherwise we return after the timeout duration with an error.
//
// This method is mostly useful in tests.
func (l *DistributedLog) WaitForLeader(timeout time.Duration) error {
	timeoutc := time.After(timeout)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutc:
			return fmt.Errorf("timed out")
		case <-ticker.C:
			// if leader has been elected
			if l.raft.Leader() != "" {
				return nil
			}
		}
	}
}

// Shutsdown the associated raft instances and closes the underlying
// commit log.
func (l *DistributedLog) Close() error {
	shutdownFuture := l.raft.Shutdown()
	if err := shutdownFuture.Error(); err != nil {
		return err
	}

	return l.log.Close()
}

// Returns a slice of all the servers in the cluster of which this
// server is a member.
func (l *DistributedLog) GetServers() ([]*api.Server, error) {
	future := l.raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return nil, err
	}

	servers := []*api.Server{}
	for _, server := range future.Configuration().Servers {
		servers = append(servers, &api.Server{
			Id:       string(server.ID),
			RpcAddr:  string(server.Address),
			IsLeader: l.raft.Leader() == server.Address,
		})
	}

	return servers, nil
}

var _ raft.FSM = (*fsm)(nil)

// Finite state machine implementation for our distributed log's
// raft instance.
type fsm struct {
	log *Log
}

// Type of request. (Enumeration of request types)
type RequestType uint8

const (
	AppendRequestType RequestType = 0
)

// Applies the given raft record to our local Raft instance. The
// record data's first byte contains the request type and the
// remainder contains the marshalled record. Based on the
// decoded record type, we apply the appropriate command.
func (l *fsm) Apply(record *raft.Log) interface{} {
	buf := record.Data
	reqType := RequestType(buf[0])
	switch reqType {
	case AppendRequestType:
		return l.applyAppend(buf[1:])
	}
	return nil
}

func (l *fsm) applyAppend(b []byte) interface{} {
	var req api.ProduceRequest
	if err := proto.Unmarshal(b, &req); err != nil {
		return err
	}

	if offset, err := l.log.Append(req.Record); err != nil {
		return err
	} else {
		return &api.ProduceResponse{Offset: offset}
	}
}

// Creates a snaphost of the data stored in the local Log instance.
// The snapshot is created from a contiguous reader over all the
// data segments in the log.
func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	return &snapshot{reader: f.log.Reader()}, nil
}

var _ raft.FSMSnapshot = (*snapshot)(nil)

// Represents a snapshot containing a io.Reader to the data to be
// stored.
type snapshot struct {
	reader io.Reader
}

// Copies the data from the underlying io.Reader to the given data
// sink.
func (s *snapshot) Persist(sink raft.SnapshotSink) error {
	if _, err := io.Copy(sink, s.reader); err != nil {
		_ = sink.Cancel()
		return err
	}
	return sink.Close()
}

func (s *snapshot) Release() {}

// Restores data from a previously stored snasphot to the local Log instance.
func (f *fsm) Restore(reader io.ReadCloser) error {
	recordSizeEncBytes := make([]byte, lenWidth)
	var recordBytesBuf bytes.Buffer

	first := true
	for {
		// read record size
		_, err := io.ReadFull(reader, recordSizeEncBytes)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		// read record bytes
		recordSize := int64(enc.Uint64(recordSizeEncBytes))
		if _, err := io.CopyN(&recordBytesBuf, reader, recordSize); err != nil {
			return err
		}

		// unmarshal record from bytes
		record := &api.Record{}
		if err := proto.Unmarshal(recordBytesBuf.Bytes(), record); err != nil {
			return err
		}

		// if this is the first record read from this reader, set this
		// log's InitialOffset to the first record's (this record's)
		// offset
		if first {
			f.log.Config.Segment.InitialOffset = record.Offset
			if err := f.log.Reset(); err != nil {
				return err
			}
		}
		first = false

		if _, err = f.log.Append(record); err != nil {
			return err
		}

		recordBytesBuf.Reset()

	}

	return nil
}

var _ raft.LogStore = (*logStore)(nil)

// Raft command log store. This store is used by the Raft instance to record
// all the commands applied to the Raft FSM.
type logStore struct{ *Log }

func newLogStore(dir string, c Config) (*logStore, error) {
	if log, err := NewLog(dir, c); err != nil {
		return nil, err
	} else {
		return &logStore{log}, nil
	}
}

func (l *logStore) FirstIndex() (uint64, error) { return l.LowestOffset() }

func (l *logStore) LastIndex() (uint64, error) { return l.HighestOffset() }

func (l *logStore) GetLog(index uint64, out *raft.Log) error {
	in, err := l.Read(index)

	if err != nil {
		return err
	}

	out.Data = in.Value
	out.Index = in.Offset
	out.Type = raft.LogType(in.Type)
	out.Term = in.Term
	return nil
}

func (l *logStore) StoreLog(record *raft.Log) error {
	return l.StoreLogs([]*raft.Log{record})
}

func (l *logStore) StoreLogs(records []*raft.Log) error {
	for _, record := range records {
		if _, err := l.Append(&api.Record{
			Value: record.Data,
			Term:  record.Term,
			Type:  uint32(record.Type),
		}); err != nil {
			return err
		}
	}

	return nil
}

func (l *logStore) DeleteRange(min, max uint64) error {
	return l.Truncate(max)
}

var _ raft.StreamLayer = (*StreamLayer)(nil)

// Raft network communication layer implementation.
type StreamLayer struct {
	ln net.Listener

	serverTLSConfig *tls.Config
	peerTLSConfig   *tls.Config
}

func NewStreamLayer(
	ln net.Listener,
	serverTLSConfig, peerTLSConfig *tls.Config,
) *StreamLayer {
	return &StreamLayer{
		ln:              ln,
		serverTLSConfig: serverTLSConfig,
		peerTLSConfig:   peerTLSConfig,
	}
}

const RaftRPC = 1

// Makes outgoing connections to other servers in the Raft cluster.
// We we connect to a server, we write a byte identifying this
// connection as a Raft RPC connection. This enables us to use the
// same port for Raft as well as Log gRPC requests.
// This method uses the peer TLS config to identiy as a client.
//
// Returns the connection made, along with an error if any.
func (s *StreamLayer) Dial(
	addr raft.ServerAddress, timeout time.Duration,
) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.Dial("tcp", string(addr))
	if err != nil {
		return nil, err
	}

	// identify to mux this is a raft rpc
	_, err = conn.Write([]byte{byte(RaftRPC)})
	if err != nil {
		return nil, err
	}

	if s.peerTLSConfig != nil {
		conn = tls.Client(conn, s.peerTLSConfig)
	}

	return conn, err
}

// Accepts incoming requests from other servers.
// This method checks if the first byte read matches the Raft RPC
// identifying byte. If it doesn't match we error our. We proceed
// normally if the byte matches.
// This method uses the server's TLS config to identify as a server.
//
// Returns the connection made along with an error if any.
func (s *StreamLayer) Accept() (net.Conn, error) {
	conn, err := s.ln.Accept()
	if err != nil {
		return nil, err
	}

	b := make([]byte, 1)
	_, err = conn.Read(b)
	if err != nil {
		return nil, err
	}

	if bytes.Compare([]byte{byte(RaftRPC)}, b) != 0 {
		return nil, fmt.Errorf("not a raft rpc")
	}
	if s.serverTLSConfig != nil {
		return tls.Server(conn, s.serverTLSConfig), nil
	}

	return conn, nil
}

func (s *StreamLayer) Close() error {
	return s.ln.Close()
}

func (s *StreamLayer) Addr() net.Addr {
	return s.ln.Addr()
}
