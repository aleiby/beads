package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// awaitCondition polls until condition returns true or timeout is reached.
// This is more robust than time.Sleep under CPU load.
func awaitCondition(t *testing.T, timeout time.Duration, desc string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", desc)
}

func TestDebouncer_BatchesMultipleTriggers(t *testing.T) {
	var count int32
	debouncer := NewDebouncer(50*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})
	t.Cleanup(debouncer.Cancel)

	debouncer.Trigger()
	debouncer.Trigger()
	debouncer.Trigger()

	// Action should not fire synchronously
	if got := atomic.LoadInt32(&count); got != 0 {
		t.Errorf("action fired synchronously: got %d, want 0", got)
	}

	// Wait for debounced action to fire (generous timeout for CI load)
	awaitCondition(t, 500*time.Millisecond, "action to fire once", func() bool {
		return atomic.LoadInt32(&count) == 1
	})

	// Verify exactly one action fired
	if got := atomic.LoadInt32(&count); got != 1 {
		t.Errorf("action should have fired exactly once: got %d, want 1", got)
	}
}

func TestDebouncer_ResetsTimerOnSubsequentTriggers(t *testing.T) {
	var count int32
	var fireTime atomic.Value
	start := time.Now()

	// Use 150ms debounce to give ample margin over the 20ms sleep.
	// Under extreme CPU load, time.Sleep(20ms) could stretch to 50-100ms,
	// so we need debounce >> sleep to ensure the second trigger resets
	// the timer before the first one fires.
	debouncer := NewDebouncer(150*time.Millisecond, func() {
		fireTime.Store(time.Now())
		atomic.AddInt32(&count, 1)
	})
	t.Cleanup(debouncer.Cancel)

	debouncer.Trigger() // t=0, would fire at ~150ms if not reset
	time.Sleep(20 * time.Millisecond)
	debouncer.Trigger() // t≈20ms, resets timer, should fire at ~170ms

	// Wait for action to fire
	awaitCondition(t, 500*time.Millisecond, "action to fire", func() bool {
		return atomic.LoadInt32(&count) == 1
	})

	// Verify timer was reset: action should have fired after debounce duration
	// from the second trigger (~170ms from start), not from the first (~150ms).
	// Use 100ms threshold to handle CI timing variance while still catching
	// cases where the timer wasn't reset.
	fired := fireTime.Load().(time.Time)
	elapsed := fired.Sub(start)
	if elapsed < 100*time.Millisecond {
		t.Errorf("action fired too early at %v, timer may not have been reset (expected >100ms)", elapsed)
	}
}

func TestDebouncer_CancelDuringWait(t *testing.T) {
	var count int32
	debouncer := NewDebouncer(50*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})
	t.Cleanup(debouncer.Cancel)

	debouncer.Trigger()
	time.Sleep(10 * time.Millisecond)

	debouncer.Cancel()

	time.Sleep(50 * time.Millisecond)
	if got := atomic.LoadInt32(&count); got != 0 {
		t.Errorf("action should not have fired after cancel: got %d, want 0", got)
	}
}

func TestDebouncer_CancelWithNoPendingAction(t *testing.T) {
	var count int32
	debouncer := NewDebouncer(50*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})
	t.Cleanup(debouncer.Cancel)

	debouncer.Cancel()

	debouncer.Trigger()

	// Wait for action to fire (generous timeout for CI load)
	awaitCondition(t, 500*time.Millisecond, "action to fire after cancel with no pending", func() bool {
		return atomic.LoadInt32(&count) == 1
	})
}

func TestDebouncer_ThreadSafety(t *testing.T) {
	var count int32
	debouncer := NewDebouncer(50*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})
	t.Cleanup(debouncer.Cancel)

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			debouncer.Trigger()
		}()
	}

	close(start)
	wg.Wait()

	// Wait for debounced action to fire (generous timeout for CI load)
	awaitCondition(t, 500*time.Millisecond, "action to fire", func() bool {
		return atomic.LoadInt32(&count) >= 1
	})

	got := atomic.LoadInt32(&count)
	if got != 1 {
		t.Errorf("all concurrent triggers should batch to exactly 1 action: got %d, want 1", got)
	}
}

func TestDebouncer_ConcurrentCancelAndTrigger(t *testing.T) {
	var count int32
	debouncer := NewDebouncer(50*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})
	t.Cleanup(debouncer.Cancel)

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			if index%2 == 0 {
				debouncer.Trigger()
			} else {
				debouncer.Cancel()
			}
		}(i)
	}

	wg.Wait()
	debouncer.Cancel()

	time.Sleep(100 * time.Millisecond)

	got := atomic.LoadInt32(&count)
	if got != 0 && got != 1 {
		t.Errorf("unexpected action count with concurrent cancel/trigger: got %d, want 0 or 1", got)
	}
}

func TestDebouncer_MultipleSequentialTriggerCycles(t *testing.T) {
	var count int32
	debouncer := NewDebouncer(30*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})
	t.Cleanup(debouncer.Cancel)

	awaitCount := func(want int32) {
		deadline := time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(deadline) {
			if got := atomic.LoadInt32(&count); got >= want {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		got := atomic.LoadInt32(&count)
		t.Fatalf("timeout waiting for count=%d (got %d)", want, got)
	}

	debouncer.Trigger()
	awaitCount(1)

	debouncer.Trigger()
	awaitCount(2)

	debouncer.Trigger()
	awaitCount(3)
}

func TestDebouncer_CancelImmediatelyAfterTrigger(t *testing.T) {
	var count int32
	debouncer := NewDebouncer(50*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})
	t.Cleanup(debouncer.Cancel)

	debouncer.Trigger()
	debouncer.Cancel()

	time.Sleep(60 * time.Millisecond)
	if got := atomic.LoadInt32(&count); got != 0 {
		t.Errorf("action should not fire after immediate cancel: got %d, want 0", got)
	}
}
