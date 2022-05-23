package main

import (
	"github.com/arindas/proglog/internal/agent"
	"github.com/arindas/proglog/internal/config"
)

func main() {
}

type cfg struct {
	agent.Config

	ServerTLSConfig, PeerTLSConfig config.TLSConfig
}
