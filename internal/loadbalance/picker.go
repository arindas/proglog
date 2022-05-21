package loadbalance

import (
	"strings"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
)

var _ base.PickerBuilder = (*Picker)(nil)

// Picker represents an entity for picking which connection to use for a request.
// It is the load balancing component of the gRPC request resolution process.
type Picker struct {
	mu sync.RWMutex

	leader balancer.SubConn

	followers []balancer.SubConn
	schedTick uint64
}

// Seperates the connections to the followers from the connection to the leader
// and stores them in a Picker instance. The picker instance built is returned.
func (p *Picker) Build(buildInfo base.PickerBuildInfo) balancer.Picker {
	p.mu.Lock()
	defer p.mu.Unlock()

	var followers []balancer.SubConn
	for sc, scInfo := range buildInfo.ReadySCs {
		isLeader := scInfo.Address.Attributes.Value("is_leader").(bool)
		if isLeader {
			p.leader = sc
			continue
		}
		followers = append(followers, sc)
	}
	p.followers = followers
	return p
}

var _ balancer.Picker = (*Picker)(nil)

// Picks a follower subconnection from the pool of available followers in a round
// robin fashion and returns it.
func (p *Picker) nextScheduledFollower() balancer.SubConn {
	// use atomic operation, as the next best mutual exclusion mechanism, since
	// this method is invoked after acquiring the lock
	tick := atomic.AddUint64(&p.schedTick, uint64(1))
	idx := int(tick % uint64(len(p.followers)))
	return p.followers[idx]
}

// Picks the subconnection to use for the given request. All writes(i.e. Produce)
// go through the leader. Read(i.e Consume) requests are balanced among followers
// in a round robin fashion.
//
// Returns the subconnection picked, along with an error if any.
func (p *Picker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result balancer.PickResult

	// if this is a write, or there are no followers, use leader
	if strings.Contains(info.FullMethodName, "Produce") || len(p.followers) == 0 {
		result.SubConn = p.leader
	} else if strings.Contains(info.FullMethodName, "Consume") {
		result.SubConn = p.nextScheduledFollower()
	}

	if result.SubConn == nil {
		return result, balancer.ErrNoSubConnAvailable
	}

	return result, nil
}

func init() {
	balancer.Register(base.NewBalancerBuilder(Name, &Picker{}, base.Config{}))
}
