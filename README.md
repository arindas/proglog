# proglog
[![Go Report Card](https://goreportcard.com/badge/github.com/arindas/proglog)](https://goreportcard.com/report/github.com/arindas/proglog)
[![ci-tests](https://github.com/arindas/proglog/actions/workflows/ci-tests.yml/badge.svg)](https://github.com/arindas/proglog/actions/workflows/ci-tests.yml)
[![mit-license](https://img.shields.io/badge/License-MIT-green.svg)](https://img.shields.io/badge/License-MIT-green.svg)

A distributed commit log.

This repository follows the book 
["Distributed Services with Go"](https://pragprog.com/titles/tjgo/distributed-services-with-go/) 
by Travis Jeffrey.

The official repository for this book can be found at https://github.com/travisjeffery/proglog 

This repository is meant for a personal record of working through the book. Progress through
different sections of the book are marked with tags. We create a new tag on the completion of a 
specific section. If you want to work on a specific section,
checkout to the previous tag, create a new branch from it and start working.

Where necessary, I have used my preferred idioms and style when implementing a specific section,
while still adhering to correctness with all the tests. Please keep this in mind if you plan to
use this as a reference. If you prefer the original source, refer to the official repository instead.

## Changelog

### v0.7.1 Chapter 7 - Replication, Log Service Agent
Implemented log replication: consume logs from every peer in the cluster and produce them locally. (This
behavior leads to infinite replication of the same record since there is no well defined leader-follower
relationship. The original producer, ends up consuming the same record from another peer which consumed
the record from itself.)

Orchestrated the different components of our log service using a single 'Agent' entity:

```go
type Agent struct {
  Config

  // … unexported members
}
```
`Agent` requires the following configuration:
```go
// Represents the configuration for our Agent.
type Config struct {
	ServerTLSConfig *tls.Config // TLS authentication config for server
	PeerTLSConfig   *tls.Config // TLS authentication config for peers

	DataDir string // Data directory for storing log records

	BindAddr       string   // Address of socket used for listening to cluster membership events
	RPCPort        int      // Port used for serving log service GRPC requests
	NodeName       string   // Node name to use for cluster membership
	StartJoinAddrs []string // Addresses of nodes from the cluster. Used for joining the cluster

	ACLModelFile  string // Access control list model file for authorization
	ACLPolicyFile string // Access control list policy file for authorization
}
```

It exposes the following methods:
```go
// RPC Socket Address with format "{BindAddrHost}:{RPCPort}"
// BindAddr and RPCAddr share the same host.
func (c Config) RPCAddr() (string, error) { … }

// Shuts down the commit log service agent. The following steps are taken: Leave Cluster, Stop record
// replication, gracefully stop RPC server, cleanup data structures for the commit log. This method
// retains the files written by the log service since they might be necessary for data recovery.
// Returns any error which occurs during the shutdown process, nil otherwise.
func (a *Agent) Shutdown() error { … }

// Constructs a new Agent instance. It take the following steps for setting up an Agent:
// Setup application logging, created data-structures for the commit log, setup the RPC
// server and finally start the cluster membership manager.
// Returns any error which occurs during the membership setup, nil otherwise.
//
// Sets up cluster membership handlers for this commit log service. This method instantiates
// the cluster membership handlers with that of the log replicator. This effectively allows this
// commit log service instance to replicate records from all nodes and any new nodes that
// joins the cluster, of which this service instance is a member. We also responsibly stop
// replicating records from any node that leaves the cluster.
func New(config Config) (*Agent, error) { … }

```

### v0.7.0 Chapter 7 - Service Discovery: Discover services with Serf
Implemented a Serf cluster membership manager. It handles cluster membership events and manages cluster
membership operations. Membership event handling is made configurable with the following interface:

```go
// Handler for cluster membership modification operations for a Node.
type Handler interface {
	Join(name, addr string) error
	Leave(name string) error
}
```

Our Membership manager used the following configuration:
```go
// Configuration of a single node in a Surf cluster.
type Config struct {
	// NodeName acts as the node's unique identifier across the Serf cluster.
	NodeName string

	// Serf listens on this address for gossiping. [Ref: gossip protocol serf]
	BindAddr string

	// Used for sharing data to other nodes in the cluster. This information is
	// used by the cluster to decide how to handle this node.
	Tags map[string]string

	// Used for joining a new node to the cluster. Joining a new node requires
	// pointing to atleast one in-cluster node. In a prod env., it's advisable
	// to specify at least 3 addrs to increase cluster resliency.
	StartJoinAddrs []string
}
```

Where "name" refers to node name and "addr" refers to the node RPC address.

On every node that it's configured in, our membership manager does the following operations:
- Binds and listen's on a unique port on localhost for mebership events from the network. (As used by
Serf's gossip protocol.)
- Sets up serf config with the binded address and port, the node name and the tags
- Routes mebership events from the network to a channel for easy consumption
- Start listening for membership events from the routed channel
- If addresses for Nodes in an existing cluster are provided via "StartJoinAddrs", we issue a join request
with the provided address to join the cluster. If not a new cluster is created with only the invoking node.

Our membership managers are created with a simple constructor function:
```go
// Creates a new Serf cluster member with the given config and cluster handler.
// It internally setups up the Serf configuration and event handlers.
func New(handler Handler, config Config) (*Membership, error) { … }

```


### v0.6.0 Chapter 6 - Observe your systems
Implemented tracing and metric collection for our service with OpenCensus. Added logging with Uber Zap.
This simply required us to setup and wire the required middlewares for each.

### v0.5.1 Chapter 5 - Corrected github action to properly install cfssl* tools.
Here's the github workflow step for generating the configs for testing authentication and authorization.

```
- name: Generate Config
  run: |
    go get github.com/cloudflare/cfssl/cmd/... 
    export PATH=${PATH}:${HOME}/go/bin
    make gencert
    make genacl
```

### v0.5.0 Chapter 5 - Secure our service
Implemented TLS authentication for clients and servers. Added authorization support with Access control lists.

```
f879aaa Tested and verified that unauthorized clients are denied access
a39869e Moved configuration necessary for tests into testconf
87fbe5f Moved to multiple clients to test ACL implementation.
44d430e Implemented mutual TLS authentication for our GRPC Log Service in tests
```

### v0.4.0 Chapter 4 - Serve Requests with gRPC
Presented the log as a gRPC service, with the following protobuf definitions:
```proto
// Messages
message Record { bytes value = 1; uint64 offset = 2; }

message ProduceRequest { Record record = 1; }
message ProduceResponse { uint64 offset = 1; }

message ConsumeRequest { uint64 offset = 1; }
message ConsumeResponse { Record record = 2; }


// Log Service
service Log {
    rpc Produce(ProduceRequest) returns (ProduceResponse) {}
    rpc Consume(ConsumeRequest) returns (ConsumeResponse) {}
    rpc ConsumeStream(ConsumeRequest) returns (stream ConsumeResponse) {}
    rpc ProduceStream(stream ProduceRequest) returns (stream ProduceResponse) {}
}
```

`Produce` and `Consume` are simply wrappers to a log implementation which produce a record and consume
a record from the underlying log.

`ProduceStream` uses a bi-directional stream. It reads records from the stream, produces them to the
log, and writes the returned offsets to the stream.

`ConsumeStream` simply accepts an offset and returns a read only stream to read records starting from
the given offset to the end of the log.
   

### v0.3.3 Chapter 3 - Completed the Log Implementation.
A log is paritioned into a collection of segments, sorted by the offsets
of the records they contain. The last segment is the active segment.

Writes goto the last segment till it's capacity is maxed out. Once it's
capacity is full, we create new segment and set it as the active segment.

Read operations are serviced by a linear search on the segments to find
the segment which contains the given offset. If the segment is found, we
simply utilize its Read() operation to read the record.

### v0.3.2 Chapter 3 - Add a segment impl for commit log package
Segment represents a Log segment with a store file and a index to speed up reads. It is
the data structure to unify data store and indexes.

The segment implementation provides the following API:
- `Append(record *api.Record) (offset uint64, err error)` 
  Appends a new record to this segment. Returns the absolute offset of
  the newly written record along with an error if any.

- `Read(off uint64) (*api.Record, error)`
  Reads the record at the given absolute offset. Returns the record read
  along with an error if any.

### v0.3.1 Chapter 3 - Add a index impl for commit log package
Implemented an index containing <segment relative offset, position in store file> entries.
It is backed by a file on disk, which is mmap()-ed for frequent access.

Real offsets are translated using segment relative offset + segment start offset. The
segment start offset is stored in the config.

The index implementation provides the following API:
- `Read(in int64) (off uint32, pos uint64, err error)`
  Reads and returns the segment relative offset and position in store for a record,
  along with an error if any.
  The parameter integer is interpreted as the sequence number. In case of negative
  values, it is interpreted from the end.
  (0 indicates first record, -1 indicates the last record)
 
- `Write(off uint32, pos uint64) error`
  Writes the given offset and the position pair at the end of the index file.
  Returns an error if any.
  
  The offset and the position are binary encoded and written at the end of
  the file. If the file doesn't have enough space to contain 12 bytes, EOF
  is returned. We also increase the size of the file by 12 bytes, post writing.

### v0.3.0 Chapter 3 - Add a store impl for commit log package
Created a buffered store implementation backed by a file for a commit log package.

The store implementation is intended to be package local and provides the following API:
- `Append([]byte) bytesWritten uint64, position uint64, err error`
  Appends the given bytes as a record to this store. Returns the number of bytes written,
  the position at which is written along with errors if any.
- `Read(pos uint64) []byte, error`
  Reads the record at the record at the given position and returns the bytes in the record
  along with an error if any.
- `Close() error`
  Closes the store by closing the underlying the file instance.

Added tests for verifying the correctness of the implementation.

### v0.2.0 Chapter 2 - Add a protobuf definition for Record
Created a protobuf definition for log record in [log.proto](./api/v1/log.proto) and generated
corresponding go stubs for it using `protoc`

Created a convenience Makefile for easily generating go stubs in the future.

### v0.1.0 Chapter 1 - Basic append only log implementation
Implemented as basic append only log and presented it with a simple REST API

The Log provides two operations:
- `Read(Offset) Record`
  Reads a record from the given offset or returns a record not found error.
- `Append(Record) Offset`
  Appends the given record at the end of the log and returns the offset at
  which it was written in the log.

We expose this log as REST API with the following methods:
- `[POST /] { record: []bytes } => { offset: uint64 }`
  Appends the record provided in json request body and returns the offset at
  which it was written in json response.
- `[GET /] { offset: uint64 } => { record: []bytes }`
  Responds with record at the offset in the request.

## License
This repository is presented under the MIT License. See [LICENSE](./LICENSE) for more details.
