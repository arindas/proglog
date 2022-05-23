package server

import (
	api "github.com/arindas/proglog/api/v1"
)

type GetServerer interface {
	GetServers() ([]*api.Server, error)
}
