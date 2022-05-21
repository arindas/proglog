package loadbalance

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
)

// Test that picker doesn't find any sub connections until resolver
// discovers servers.
func TestPickerNoSubConnAvailable(t *testing.T) {
	picker := &Picker{}
	for _, method := range []string{
		"/log.vX.Log/Produce",
		"/log.vX.Log/Consume",
	} {
		info := balancer.PickInfo{FullMethodName: method}
		result, err := picker.Pick(info)
		require.Equal(t, err, balancer.ErrNoSubConnAvailable)
		require.Nil(t, result.SubConn)
	}
}

var _ balancer.SubConn = (*subConn)(nil)

type subConn struct{ addrs []resolver.Address }

func (s *subConn) UpdateAddresses(addrs []resolver.Address) {
	s.addrs = addrs
}

func (s *subConn) Connect() {}

func setupTest() (*Picker, []*subConn) {
	subConns := []*subConn{}
	pickerBuildInfo := base.PickerBuildInfo{
		ReadySCs: map[balancer.SubConn]base.SubConnInfo{},
	}

	// 3 member cluster, with 0th member as leader
	for i := 0; i < 3; i++ {
		sc := &subConn{}
		addr := resolver.Address{
			Attributes: attributes.New("is_leader", i == 0),
		}
		sc.UpdateAddresses([]resolver.Address{addr})

		pickerBuildInfo.ReadySCs[sc] = base.SubConnInfo{Address: addr}
		subConns = append(subConns, sc)
	}

	picker := &Picker{}
	picker.Build(pickerBuildInfo)

	return picker, subConns
}

func TestPickerProducesToLeader(t *testing.T) {
	picker, subConns := setupTest()
	pickInfo := balancer.PickInfo{
		FullMethodName: "/log.vX.Log/Produce",
	}

	for i := 0; i < 5; i++ {
		gotPick, err := picker.Pick(pickInfo)
		require.NoError(t, err)
		require.Equal(t, gotPick.SubConn, subConns[0])
	}
}

func TestPickerConsumesFromFollowers(t *testing.T) {
	picker, subConns := setupTest()
	pickInfo := balancer.PickInfo{
		FullMethodName: "/log.vX.Log/Consume",
	}

	for i := 0; i < 5; i++ {
		gotPick, err := picker.Pick(pickInfo)
		require.NoError(t, err)
		require.Equal(t, gotPick.SubConn, subConns[i%2+1])
	}
}
