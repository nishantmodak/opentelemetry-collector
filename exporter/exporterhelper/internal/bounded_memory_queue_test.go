// Copyright The OpenTelemetry Authors
// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
)

func newNopQueueSettings(callback func(item Request)) QueueSettings {
	return QueueSettings{
		DataType: component.DataTypeMetrics,
		Callback: callback,
	}
}

type stringRequest struct {
	Request
	str string
}

func newStringRequest(str string) Request {
	return stringRequest{str: str}
}

// In this test we run a queue with capacity 1 and a single consumer.
// We want to test the overflow behavior, so we block the consumer
// by holding a startLock before submitting items to the queue.
func TestBoundedQueue(t *testing.T) {
	q := NewBoundedMemoryQueue(1, 1)

	var startLock sync.Mutex

	startLock.Lock() // block consumers
	consumerState := newConsumerState(t)

	assert.NoError(t, q.Start(context.Background(), componenttest.NewNopHost(), newNopQueueSettings(func(item Request) {
		consumerState.record(item.(stringRequest).str)

		// block further processing until startLock is released
		startLock.Lock()
		//nolint:staticcheck // SA2001 ignore this!
		startLock.Unlock()
	})))

	assert.True(t, q.Produce(newStringRequest("a")))

	// at this point "a" may or may not have been received by the consumer go-routine
	// so let's make sure it has been
	consumerState.waitToConsumeOnce()

	// at this point the item must have been read off the queue, but the consumer is blocked
	assert.Equal(t, 0, q.Size())
	consumerState.assertConsumed(map[string]bool{
		"a": true,
	})

	// produce two more items. The first one should be accepted, but not consumed.
	assert.True(t, q.Produce(newStringRequest("b")))
	assert.Equal(t, 1, q.Size())
	// the second should be rejected since the queue is full
	assert.False(t, q.Produce(newStringRequest("c")))
	assert.Equal(t, 1, q.Size())

	startLock.Unlock() // unblock consumer

	consumerState.assertConsumed(map[string]bool{
		"a": true,
		"b": true,
	})

	// now that consumers are unblocked, we can add more items
	expected := map[string]bool{
		"a": true,
		"b": true,
	}
	for _, item := range []string{"d", "e", "f"} {
		assert.True(t, q.Produce(newStringRequest(item)))
		expected[item] = true
		consumerState.assertConsumed(expected)
	}

	assert.NoError(t, q.Shutdown(context.Background()))
	assert.False(t, q.Produce(newStringRequest("x")), "cannot push to closed queue")
}

// In this test we run a queue with many items and a slow consumer.
// When the queue is stopped, the remaining items should be processed.
// Due to the way q.Stop() waits for all consumers to finish, the
// same lock strategy use above will not work, as calling Unlock
// only after Stop will mean the consumers are still locked while
// trying to perform the final consumptions.
func TestShutdownWhileNotEmpty(t *testing.T) {
	q := NewBoundedMemoryQueue(10, 1)

	consumerState := newConsumerState(t)

	assert.NoError(t, q.Start(context.Background(), componenttest.NewNopHost(), newNopQueueSettings(func(item Request) {
		consumerState.record(item.(stringRequest).str)
		time.Sleep(1 * time.Second)
	})))

	q.Produce(newStringRequest("a"))
	q.Produce(newStringRequest("b"))
	q.Produce(newStringRequest("c"))
	q.Produce(newStringRequest("d"))
	q.Produce(newStringRequest("e"))
	q.Produce(newStringRequest("f"))
	q.Produce(newStringRequest("g"))
	q.Produce(newStringRequest("h"))
	q.Produce(newStringRequest("i"))
	q.Produce(newStringRequest("j"))

	assert.NoError(t, q.Shutdown(context.Background()))

	assert.False(t, q.Produce(newStringRequest("x")), "cannot push to closed queue")
	consumerState.assertConsumed(map[string]bool{
		"a": true,
		"b": true,
		"c": true,
		"d": true,
		"e": true,
		"f": true,
		"g": true,
		"h": true,
		"i": true,
		"j": true,
	})
	assert.Equal(t, 0, q.Size())
}

func Benchmark_QueueUsage_10000_1_50000(b *testing.B) {
	queueUsage(b, 10000, 1, 50000)
}

func Benchmark_QueueUsage_10000_2_50000(b *testing.B) {
	queueUsage(b, 10000, 2, 50000)
}
func Benchmark_QueueUsage_10000_5_50000(b *testing.B) {
	queueUsage(b, 10000, 5, 50000)
}
func Benchmark_QueueUsage_10000_10_50000(b *testing.B) {
	queueUsage(b, 10000, 10, 50000)
}

func Benchmark_QueueUsage_50000_1_50000(b *testing.B) {
	queueUsage(b, 50000, 1, 50000)
}

func Benchmark_QueueUsage_50000_2_50000(b *testing.B) {
	queueUsage(b, 50000, 2, 50000)
}
func Benchmark_QueueUsage_50000_5_50000(b *testing.B) {
	queueUsage(b, 50000, 5, 50000)
}
func Benchmark_QueueUsage_50000_10_50000(b *testing.B) {
	queueUsage(b, 50000, 10, 50000)
}

func Benchmark_QueueUsage_10000_1_250000(b *testing.B) {
	queueUsage(b, 10000, 1, 250000)
}

func Benchmark_QueueUsage_10000_2_250000(b *testing.B) {
	queueUsage(b, 10000, 2, 250000)
}
func Benchmark_QueueUsage_10000_5_250000(b *testing.B) {
	queueUsage(b, 10000, 5, 250000)
}
func Benchmark_QueueUsage_10000_10_250000(b *testing.B) {
	queueUsage(b, 10000, 10, 250000)
}

func queueUsage(b *testing.B, capacity int, numConsumers int, numberOfItems int) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q := NewBoundedMemoryQueue(capacity, numConsumers)
		err := q.Start(context.Background(), componenttest.NewNopHost(), newNopQueueSettings(func(item Request) {
			time.Sleep(1 * time.Millisecond)
		}))
		require.NoError(b, err)
		for j := 0; j < numberOfItems; j++ {
			q.Produce(newStringRequest(fmt.Sprintf("%d", j)))
		}
		assert.NoError(b, q.Shutdown(context.Background()))
	}
}

type consumerState struct {
	sync.Mutex
	t            *testing.T
	consumed     map[string]bool
	consumedOnce *atomic.Bool
}

func newConsumerState(t *testing.T) *consumerState {
	return &consumerState{
		t:            t,
		consumed:     make(map[string]bool),
		consumedOnce: &atomic.Bool{},
	}
}

func (s *consumerState) record(val string) {
	s.Lock()
	defer s.Unlock()
	s.consumed[val] = true
	s.consumedOnce.Store(true)
}

func (s *consumerState) snapshot() map[string]bool {
	s.Lock()
	defer s.Unlock()
	out := make(map[string]bool)
	for k, v := range s.consumed {
		out[k] = v
	}
	return out
}

func (s *consumerState) waitToConsumeOnce() {
	require.Eventually(s.t, s.consumedOnce.Load, 2*time.Second, 10*time.Millisecond, "expected to consumer once")
}

func (s *consumerState) assertConsumed(expected map[string]bool) {
	for i := 0; i < 1000; i++ {
		if snapshot := s.snapshot(); !reflect.DeepEqual(snapshot, expected) {
			time.Sleep(time.Millisecond)
		}
	}
	assert.Equal(s.t, expected, s.snapshot())
}

func TestZeroSizeWithConsumers(t *testing.T) {
	q := NewBoundedMemoryQueue(0, 1)

	err := q.Start(context.Background(), componenttest.NewNopHost(), newNopQueueSettings(func(item Request) {}))
	assert.NoError(t, err)

	assert.True(t, q.Produce(newStringRequest("a"))) // in process

	assert.NoError(t, q.Shutdown(context.Background()))
}

func TestZeroSizeNoConsumers(t *testing.T) {
	q := NewBoundedMemoryQueue(0, 0)

	err := q.Start(context.Background(), componenttest.NewNopHost(), newNopQueueSettings(func(item Request) {}))
	assert.NoError(t, err)

	assert.False(t, q.Produce(newStringRequest("a"))) // in process

	assert.NoError(t, q.Shutdown(context.Background()))
}
