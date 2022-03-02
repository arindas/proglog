# proglog
A distributed commit log.


## Changelog

### v0.1.0 Chapter 1 - Basic append only log implementation
Implemented as basic append only log and presented it with a simple REST API

The Log provides two operations:
- Read(Offset) Record
  Reads a record from the given offset or returns a record not found error.
- Append(Record) Offset
  Appends the given record at the end of the log and returns the offset at
  which it was written in the log.

We expose this log as REST API with the following methods:
- [POST /] { record: []bytes } => { offset: uint64 }
  Appends the record provided in json request body and returns the offset at
  which it was written in json response.
- [GET /] { offset: uint64 } => { record: []bytes }
  Responds with record at the offset in the request.
