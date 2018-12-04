package p2p

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBroadcast(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	runP2P := func(t *testing.T, options ...Option) {
		ctx := context.Background()
		n := 10
		hosts := make([]*Host, n)
		for i := 0; i < n; i++ {
			opts := []Option{
				Port(30000 + i),
				SecureIO(),
			}
			opts = append(opts, options...)
			host, err := NewHost(ctx, opts...)
			require.NoError(t, err)
			require.NoError(t, host.AddBroadcastPubSub("test", func(data []byte) error {
				fmt.Print(string(data))
				fmt.Printf(", received by %s\n", host.Identity())
				return nil
			}))
			hosts[i] = host
		}

		bootstrapAddr := hosts[0].Address()
		for i := 0; i < n; i++ {
			if i != 0 {
				require.NoError(t, hosts[i].Connect(bootstrapAddr))
			}
			require.NoError(t, hosts[i].JoinOverlay())
		}

		for i := 0; i < n; i++ {
			require.NoError(
				t,
				hosts[i].Broadcast("test", []byte(fmt.Sprintf("msg sent from %s", hosts[i].Identity()))),
			)
		}

		time.Sleep(5 * time.Second)

		for i := 0; i < n; i++ {
			require.NoError(t, hosts[i].Close())
		}
	}

	t.Run("flood", func(t *testing.T) {
		runP2P(t)
	})

	t.Run("gossip", func(t *testing.T) {
		runP2P(t, Gossip())
	})
}

func TestUnicast(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	n := 10
	hosts := make([]*Host, n)
	for i := 0; i < n; i++ {
		host, err := NewHost(ctx, Port(30000+i), SecureIO())
		require.NoError(t, err)
		require.NoError(t, host.AddUnicastPubSub("test", func(data []byte) error {
			fmt.Print(string(data))
			fmt.Printf(", received by %s\n", host.Identity())
			return nil
		}))
		hosts[i] = host
	}

	bootstrapAddr := hosts[0].Address()
	for i := 0; i < n; i++ {
		if i != 0 {
			require.NoError(t, hosts[i].Connect(bootstrapAddr))
		}
		require.NoError(t, hosts[i].JoinOverlay())
	}

	for i, host := range hosts {
		neighbors, err := host.Neighbors()
		require.NoError(t, err)
		require.True(t, len(neighbors) > 0)

		for _, neighbor := range neighbors {
			require.NoError(
				t,
				host.Unicast(neighbor, "test", []byte(fmt.Sprintf("msg sent from %s", hosts[i].Identity()))),
			)
		}
	}
}
