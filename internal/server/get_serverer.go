package server

import (
	api "github.com/arindas/proglog/api/log_v1"
)

type GetServerer interface {
	GetServers() ([]*api.Server, error)
}
