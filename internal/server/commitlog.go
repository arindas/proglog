package server

import (
	api "github.com/arindas/proglog/api/log_v1"
)

type CommitLog interface {
	Append(*api.Record) (uint64, error)
	Read(uint64) (*api.Record, error)
}
