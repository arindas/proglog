package agent

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	api "github.com/arindas/proglog/api/log_v1"
	"github.com/arindas/proglog/internal/auth"
	"github.com/arindas/proglog/internal/discovery"
	"github.com/arindas/proglog/internal/log"
	"github.com/arindas/proglog/internal/server"
)

// Agent orchestrates the different components of our commit log service. It runs on every instance of our service.
type Agent struct {
	Config

	log        *log.Log
	server     *grpc.Server
	membership *discovery.Membership
	replicator *log.Replicator

	shutdown     bool
	shutdowns    chan struct{} // provision for signalling any running go routines; unutilized as of now
	shutdownLock sync.Mutex
}

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

// RPC Socket Address with format "{BindAddrHost}:{RPCPort}"
// BindAddr and RPCAddr share the same host.
func (c Config) RPCAddr() (string, error) {
	if host, _, err := net.SplitHostPort(c.BindAddr); err != nil {
		return "", err
	} else {
		return fmt.Sprintf("%s:%d", host, c.RPCPort), nil
	}
}

// Shuts down the commit log service agent. The following steps are taken: Leave Cluster, Stop record
// replication, gracefully stop RPC server, cleanup data structures for the commit log. This method
// retains the files written by the log service since they might be necessary for data recovery.
// Returns any error which occurs during the shutdown process, nil otherwise.
func (a *Agent) Shutdown() error {
	a.shutdownLock.Lock()
	defer a.shutdownLock.Unlock()

	if a.shutdown {
		return nil
	}
	a.shutdown = true
	close(a.shutdowns)

	shutdownFns := []func() error{
		a.membership.Leave,
		a.replicator.Close,
		func() error {
			a.server.GracefulStop()
			return nil
		},
		a.log.Close,
	}

	for _, fn := range shutdownFns {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}

func (a *Agent) setupLogger() error {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return err
	}
	zap.ReplaceGlobals(logger)
	return nil
}

func (a *Agent) setupLog() error {
	var err error
	a.log, err = log.NewLog(a.Config.DataDir, log.Config{})
	return err
}

func (a *Agent) setupServer() error {
	authorizer := auth.New(a.Config.ACLModelFile, a.Config.ACLPolicyFile)
	serverConfig := &server.Config{CommitLog: a.log, Authorizer: authorizer}

	grpcServerOpts := []grpc.ServerOption{}
	if a.Config.ServerTLSConfig != nil {
		creds := credentials.NewTLS(a.Config.ServerTLSConfig)
		grpcServerOpts = append(grpcServerOpts, grpc.Creds(creds))
	}

	var err error
	a.server, err = server.NewGRPCServer(serverConfig, grpcServerOpts...)
	if err != nil {
		return err
	}

	rpcAddr, err := a.RPCAddr()
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", rpcAddr)
	if err != nil {
		return err
	}

	go func(server *grpc.Server) {
		if err := server.Serve(ln); err != nil {
			_ = a.Shutdown()
		}

	}(a.server)

	return nil
}

// Sets up cluster membership handlers for this commit log service. This method instantiates
// the cluster membership handlers with that of the log replicator. This effectively allows this
// commit log service instance to replicate records from all nodes and any new nodes that
// joins the cluster, of which this service instance is a member. We also responsibly stop
// replicating records from any node that leaves the cluster.
// Returns any error which occurs during the membership setup, nil otherwise.
func (a *Agent) setupMembership() error {
	rpcAddr, err := a.Config.RPCAddr()
	if err != nil {
		return err
	}

	grpcDialOpts := []grpc.DialOption{}
	if a.Config.PeerTLSConfig != nil {
		grpcDialOpts = append(
			grpcDialOpts,
			grpc.WithTransportCredentials(credentials.NewTLS(a.Config.PeerTLSConfig)),
		)
	}

	conn, err := grpc.Dial(rpcAddr, grpcDialOpts...)
	if err != nil {
		return err
	}
	client := api.NewLogClient(conn)

	a.replicator = &log.Replicator{DialOptions: grpcDialOpts, LocalServer: client}

	a.membership, err = discovery.New(a.replicator, discovery.Config{
		NodeName: a.Config.NodeName,
		BindAddr: a.Config.BindAddr,
		Tags: map[string]string{
			"rpc_addr": rpcAddr,
		},
		StartJoinAddrs: a.Config.StartJoinAddrs,
	})

	return err
}

// Constructs a new Agent instance. It take the following steps for setting up an Agent:
// Setup application logging, created data-structures for the commit log, setup the RPC
// server and finally start the cluster membership manager.
// Returns any error which occurs during the membership setup, nil otherwise.
func New(config Config) (*Agent, error) {
	agent := &Agent{Config: config, shutdowns: make(chan struct{})}

	setupFns := []func() error{
		agent.setupLogger,
		agent.setupLog,
		agent.setupServer,
		agent.setupMembership,
	}

	for _, fn := range setupFns {
		if err := fn(); err != nil {
			return nil, err
		}
	}

	return agent, nil
}
