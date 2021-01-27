package clock

import (
	"sync"
	"time"
)

var _ Clock = (*FakeClock)(nil)

type waiter struct {
	timeout  time.Duration
	notifyCh chan time.Time

	// Set to true when the channel for this waiter has been returned to the
	// timeout/timer consumer.
	consumerWaiting bool
}

// FakeClock is a clock implementation that allows time to be programmatically
// advanced via calls to its Advance/WaitAdvance methods. It is meant to be
// used as a drop-in replacement for the wall clock in tests.
type FakeClock struct {
	mu      sync.Mutex
	waiters []*waiter
	curTime time.Time
}

// NewFakeClock creates a new fake clock instance whose time is set to curTime.
func NewFakeClock(curTime time.Time) *FakeClock {
	return &FakeClock{curTime: curTime}
}

// Now returns the current time.
func (fc *FakeClock) Now() time.Time {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.curTime
}

// After waits for the duration to elapse and then sends the current time on
// the returned channel.
func (fc *FakeClock) After(d time.Duration) <-chan time.Time {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	waiter := makeWaiter(d, true)
	fc.waiters = append(fc.waiters, waiter)
	return waiter.notifyCh
}

// NewTimer creates a new Timer that will send the current time on its
// channel after at least duration d.
func (fc *FakeClock) NewTimer(d time.Duration) Timer {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Timers are similar to waiters with the exception that the consumerWaiting
	// flag will only be set once the timer's C() method has been invoked.
	waiter := makeWaiter(d, false)
	fc.waiters = append(fc.waiters, waiter)
	return fakeClockTimer{fc: fc, waiter: waiter}
}

// WaitAdvance blocks until at least the requested number of waiters has
// received back a timeout/timer channel from the clock and then advances the
// clock by d.
func (fc *FakeClock) WaitAdvance(numWaiters int, d time.Duration) {
	for {
		fc.mu.Lock()
		var waitingConsumers int
		for _, w := range fc.waiters {
			if w.consumerWaiting {
				waitingConsumers++
			}

			if waitingConsumers == numWaiters {
				fc.mu.Unlock()
				fc.Advance(d)
				return
			}
		}
		fc.mu.Unlock()

		// Poll a bit later
		<-time.After(100 * time.Millisecond)
	}
}

// Advance the clock by d and handle expired timers/timeouts.
func (fc *FakeClock) Advance(d time.Duration) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.curTime.Add(d)

	// Notify and remove expired waiters.
	var activeWaiters []*waiter
	for _, waiter := range fc.waiters {
		if waiter.timeout <= d {
			waiter.timeout = 0
			select {
			case waiter.notifyCh <- fc.curTime:
			default:
				// Consumer has not received the last
				// notification yet.  Drop the new one to the
				// floor.
			}
			continue
		}
		waiter.timeout -= d
		activeWaiters = append(activeWaiters, waiter)
	}
	fc.waiters = activeWaiters
}

func makeWaiter(d time.Duration, consumerWaiting bool) *waiter {
	return &waiter{
		timeout:         d,
		notifyCh:        make(chan time.Time, 1),
		consumerWaiting: consumerWaiting,
	}
}

type fakeClockTimer struct {
	fc     *FakeClock
	waiter *waiter
}

func (ft fakeClockTimer) C() <-chan time.Time {
	ft.fc.mu.Lock()
	defer ft.fc.mu.Unlock()

	ft.waiter.consumerWaiting = true
	return ft.waiter.notifyCh
}

func (ft fakeClockTimer) Reset(d time.Duration) {
	ft.fc.mu.Lock()
	defer ft.fc.mu.Unlock()
	ft.waiter.timeout = d
}

func (ft fakeClockTimer) Stop() bool {
	ft.fc.mu.Lock()
	defer ft.fc.mu.Unlock()

	alreadyFired := ft.waiter.timeout == 0
	var activeWaiters []*waiter
	for _, w := range ft.fc.waiters {
		if w == ft.waiter {
			continue
		}
		activeWaiters = append(activeWaiters, w)
	}
	ft.fc.waiters = activeWaiters

	return alreadyFired
}
