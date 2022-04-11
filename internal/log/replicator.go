package log

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	api "github.com/arindas/proglog/api/log_v1"
)

type Replicator struct {
	DialOptions []grpc.DialOption
	LocalServer api.LogClient

	logger *zap.Logger

	mu      sync.Mutex
	servers map[string]chan struct{}
	closed  bool
	close   chan struct{}
}

func (r *Replicator) init() {
	if r.logger == nil {
		r.logger = zap.L().Named("replicator")
	}
	if r.servers == nil {
		r.servers = make(map[string]chan struct{})
	}
	if r.close == nil {
		r.close = make(chan struct{})
	}
}

func (r *Replicator) logError(err error, msg, addr string) {
	r.logger.Error(msg, zap.String("addr", addr), zap.Error(err))
}

func (r *Replicator) replicate(addr string, leave <-chan struct{}) {
	cc, err := grpc.Dial(addr, r.DialOptions...)
	if err != nil {
		r.logError(err, "failed to dial", addr)
		return
	}
	defer cc.Close()

	ctx := context.Background()
	client := api.NewLogClient(cc)

	// consume all the messages from the start from peer
	stream, err := client.ConsumeStream(ctx, &api.ConsumeRequest{Offset: 0})

	if err != nil {
		r.logError(err, "failed to consume", addr)
		return
	}

	// consume records from peer and produce in local server concurrently

	// used as a queue
	records := make(chan *api.Record)

	replicatorDone := make(chan struct{})
	defer close(replicatorDone)

	// consume records concurrently from peer
	go func(done <-chan struct{}) {
		for {
			select {
			case <-done:
				return
			default:
				recv, err := stream.Recv()
				if err != nil {
					r.logError(err, "failed to receive", addr)
					return
				}
				records <- recv.Record
			}
		}
	}(replicatorDone)

	// produce records consumed from peer on the local server
	for {
		select {
		case <-r.close:
			return // replicator instance closed; return
		case <-leave:
			return // server left; stop replicating
		case record := <-records:
			if _, err = r.LocalServer.Produce(ctx, &api.ProduceRequest{
				Record: record,
			}); err != nil {
				return
			}
		}
	}

}

// "Joins" or adds this server to the list of server to replicate records
// from. It starts off record replication from this server in a goroutine
// and returns.
func (r *Replicator) Join(name, addr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.init()

	if r.closed {
		return nil
	}

	if _, ok := r.servers[name]; ok {
		// already replicating this server; skip
		return nil
	}
	r.servers[name] = make(chan struct{})

	go r.replicate(addr, r.servers[name])

	return nil
}

// Removes this server from the list of servers to replicate records from.
// Also signals the replicating goroutine for this server to return.
func (r *Replicator) Leave(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.init()

	if _, ok := r.servers[name]; !ok {
		// server not being replicated; nop
		return nil
	}

	// signal that this server has left
	// to stop replication for this server
	close(r.servers[name])

	// remove entry for this server
	delete(r.servers, name)

	return nil
}

func (r *Replicator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.init()

	if r.closed {
		return nil
	}

	r.closed = true
	// signal all replicating goroutines to return
	close(r.close)

	return nil
}
