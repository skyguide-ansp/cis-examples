package util_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/skyguide-ansp/cis-examples/util"
	"github.com/stretchr/testify/require"
)

func TestConcurrentCollector_MergeStreams(t *testing.T) {
	c := &util.ConcurrentCollector[int]{}

	err := c.RegisterStreamer(func(ctx context.Context, out chan *int) {
		v := 1
		out <- &v
	})
	require.NoError(t, err)

	err = c.RegisterStreamer(func(ctx context.Context, out chan *int) {
		v := 2
		out <- &v
	})
	require.NoError(t, err)

	ctx := context.Background()
	ch, err := c.Run(ctx)
	require.NoError(t, err)

	values := make(map[int]bool)
	for v := range ch {
		values[*v] = true
	}

	require.Len(t, values, 2)
	require.True(t, values[1])
	require.True(t, values[2])
}

func TestConcurrentCollector_ClosesChannel(t *testing.T) {
	c := &util.ConcurrentCollector[int]{}

	err := c.RegisterStreamer(func(ctx context.Context, out chan *int) {})
	require.NoError(t, err)
	err = c.RegisterStreamer(func(ctx context.Context, out chan *int) {})
	require.NoError(t, err)

	ch, err := c.Run(context.Background())
	require.NoError(t, err)

	_, ok := <-ch
	require.False(t, ok)
}

func TestConcurrentCollector_ContextCancel(t *testing.T) {
	c := &util.ConcurrentCollector[int]{}

	err := c.RegisterStreamer(func(ctx context.Context, out chan *int) {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
			t.Fatal("did not respect context cancellation")
		}
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := c.Run(ctx)
	require.NoError(t, err)

	cancel()

	select {
	case _, ok := <-ch:
		require.False(t, ok)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for channel close")
	}
}

func TestConcurrentCollector_AddFuncWhileRunning(t *testing.T) {
	c := &util.ConcurrentCollector[int]{}

	block := make(chan struct{})

	err := c.RegisterStreamer(func(ctx context.Context, out chan *int) {
		<-block
	})
	require.NoError(t, err)

	_, err = c.Run(context.Background())
	require.NoError(t, err)

	err = c.RegisterStreamer(func(ctx context.Context, out chan *int) {})
	require.Error(t, err)

	close(block)
}

func TestConcurrentCollector_NoFunctions(t *testing.T) {
	c := &util.ConcurrentCollector[int]{}

	_, err := c.Run(context.Background())
	require.Error(t, err)
}

func TestConcurrentCollector_ConcurrentWriters(t *testing.T) {
	c := &util.ConcurrentCollector[int]{}

	const n = 100
	err := c.RegisterStreamer(func(ctx context.Context, out chan *int) {
		var wg sync.WaitGroup
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(v int) {
				defer wg.Done()
				out <- &v
			}(i)
		}
		wg.Wait()
	})
	require.NoError(t, err)

	ch, err := c.Run(context.Background())
	require.NoError(t, err)

	count := 0
	for range ch {
		count++
	}

	if count != n {
		t.Fatalf("expected %d values, got %d", n, count)
	}
}
