package discovery

import (
	"net"

	"github.com/hashicorp/serf/serf"
	"go.uber.org/zap"
)

// Configuration of a single node in a Surf cluster.
type Config struct {
	// NodeName acts as the node's unique identifier across the Serf cluster.
	NodeName string

	// Serf listens on this address for gossiping. [Ref: gossip protocol serf]
	BindAddr string

	// Used for sharing data to other nodes in the cluster. This information is
	// used by the cluster to decide how to handle this node.
	Tags map[string]string

	// Used for joining a new node to the cluster. Joining a new node requires
	// pointing to atleast one in-cluster node. In a prod env., it's advisable
	// to specify at least 3 addrs to increase cluster resliency.
	StartJoinAddrs []string
}

// Handler for cluster membership modification operations.
type Handler interface {
	Join(name, addr string) error
	Leave(name string) error
}

// Represents membership of a node to a cluster. It wraps a Serf instance, providing
// a handle to cluster membership operations.
type Membership struct {
	Config

	handler Handler
	serf    *serf.Serf
	events  chan serf.Event
	logger  *zap.Logger
}

// Returns true if the given Serf member refers to the local member (i.e. the
// invoking node.) by having the same name.
func (m *Membership) isLocal(member serf.Member) bool {
	return m.serf.LocalMember().Name == member.Name
}

// Returns a point-in-time snapshot of the cluster's Serf members.
func (m *Membership) Members() []serf.Member { return m.serf.Members() }

// Telles this member to leave the Serf cluster.
func (m *Membership) Leave() error { return m.serf.Leave() }

func (m *Membership) logError(err error, msg string, member serf.Member) {
	m.logger.Error(
		msg,
		zap.Error(err),
		zap.String("name", member.Name),
		zap.String("rpc_addr", member.Tags["rpc_addr"]),
	)
}

func (m *Membership) handleJoin(member serf.Member) {
	if err := m.handler.Join(member.Name, member.Tags["rpc_addr"]); err != nil {
		m.logError(err, "failed to join", member)
	}
}

func (m *Membership) handleLeave(member serf.Member) {
	if err := m.handler.Leave(member.Name); err != nil {
		m.logError(err, "failed to leave", member)
	}
}

func (m *Membership) eventHandler() {
	for event := range m.events {
		eventMembers := event.(serf.MemberEvent).Members

		switch event.EventType() {
		case serf.EventMemberJoin:
			for _, member := range eventMembers {
				if m.isLocal(member) {
					continue
				}

				m.handleJoin(member)
			}

		case
			serf.EventMemberLeave,
			serf.EventMemberFailed:
			for _, member := range eventMembers {
				if m.isLocal(member) {
					return
				}

				m.handleLeave(member)
			}
		}

	}
}

func (m *Membership) setupSerf() (err error) {
	addr, err := net.ResolveTCPAddr("tcp", m.BindAddr)
	if err != nil {
		return err
	}

	config := serf.DefaultConfig()
	config.Init()
	config.MemberlistConfig.BindAddr = addr.IP.String()
	config.MemberlistConfig.BindPort = addr.Port
	m.events = make(chan serf.Event)
	config.EventCh = m.events
	config.Tags = m.Tags
	config.NodeName = m.Config.NodeName

	m.serf, err = serf.Create(config)
	if err != nil {
		return err
	}

	go m.eventHandler()

	if m.StartJoinAddrs != nil {
		if _, err := m.serf.Join(m.StartJoinAddrs, true); err != nil {
			return err
		}
	}

	return nil
}

// Creates a new Serf cluster member with the given config and cluster handler.
// It internally setups up the Serf configuration and event handlers.
func New(handler Handler, config Config) (*Membership, error) {
	membership := &Membership{
		Config:  config,
		handler: handler,
		logger:  zap.L().Named("Membership"),
	}

	if err := membership.setupSerf(); err != nil {
		return nil, err
	}

	return membership, nil
}
