package util

import (
	"context"
	"errors"
	"sync"
)

type Streamer[T any] func(ctx context.Context, channel chan *T)

type ConcurrentCollector[T any] struct {
	lock      sync.Mutex
	funcs     []Streamer[T]
	isRunning bool
}

// RegisterStreamer registers a function to be executed, given the output channel that will contain the merged stream
func (collector *ConcurrentCollector[T]) RegisterStreamer(f Streamer[T]) error {
	collector.lock.Lock()
	defer collector.lock.Unlock()

	if collector.isRunning {
		return errors.New("can't add when running")
	}

	collector.funcs = append(collector.funcs, f)

	return nil
}

// Run execute all given function in background and provide a channel that will contains all
// the data from all the source, the collector will close the channel once every streamer return
// Streamer functions are expected to return when context is Done
func (collector *ConcurrentCollector[T]) Run(ctx context.Context) (chan *T, error) {

	// check running and block addFunc
	collector.lock.Lock()
	if collector.isRunning {
		collector.lock.Unlock()
		return nil, errors.New("already running")
	}
	collector.isRunning = true
	snapshot := append([]Streamer[T](nil), collector.funcs...)
	collector.lock.Unlock()

	if len(collector.funcs) == 0 {
		return nil, errors.New("no function to run")
	}

	// create output channel
	out := make(chan *T)

	var wg sync.WaitGroup

	wg.Add(len(snapshot))

	for _, current := range snapshot {
		go func(streamer Streamer[T]) {
			defer wg.Done()

			// Context cancellation is acceptable
			streamer(ctx, out)
		}(current)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out, nil
}
