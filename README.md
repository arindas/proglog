# proglog
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
