package log

import (
	"github.com/hashicorp/raft"
)

type Config struct {
	Raft struct {
		raft.Config
		StreamLayer *StreamLayer
		BootStrap   bool
	}
	Segment struct {
		MaxStoreBytes uint64
		MaxIndexBytes uint64
		InitialOffset uint64
	}
}
