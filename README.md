# proglog
[![ci-tests](https://github.com/arindas/proglog/actions/workflows/ci-tests.yml/badge.svg)](https://github.com/arindas/proglog/actions/workflows/ci-tests.yml)

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
