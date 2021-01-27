package clock

import (
	"time"

	gc "gopkg.in/check.v1"
)

var _ = gc.Suite(&fakeClockSuite{})

type fakeClockSuite struct {
}

func (fakeClockSuite) TestGetCurrentTime(c *gc.C) {
	now := time.Now()
	clk := NewFakeClock(now)
	c.Assert(clk.Now(), gc.DeepEquals, now)

	// Advance clock and ensure that the current time gets correctly updated.
	advance := 90 * time.Second
	now = now.Add(advance)
	clk.Advance(advance)
}

func (fakeClockSuite) TestAdvanceTriggersTimeout(c *gc.C) {
	clk := NewFakeClock(time.Now())

	timeout1Ch := clk.After(10 * time.Minute)
	timeout2Ch := clk.After(11 * time.Minute)

	go func() {
		// This should trigger both timeouts to fire.
		clk.Advance(11 * time.Minute)
	}()

	select {
	case <-timeout1Ch:
	case <-time.After(3 * time.Second):
		c.Error("timeout waiting for fake clock to trigger 10 min timeout after async advance")
	}

	select {
	case <-timeout2Ch:
	case <-time.After(3 * time.Second):
		c.Error("timeout waiting for fake clock to trigger 11 min timeout after async advance")
	}

	clk.mu.Lock()
	numWaiters := len(clk.waiters)
	clk.mu.Unlock()
	c.Assert(numWaiters, gc.Equals, 0, gc.Commentf("expired waiters were not removed from clock waiter list"))
}

func (fakeClockSuite) TestTimer(c *gc.C) {
	clk := NewFakeClock(time.Now())
	timer := clk.NewTimer(10 * time.Minute)

	// Advance the clock to trigger the timer. Then try to stop the timer
	// and check that it reports back that it has already fired.
	clk.Advance(10 * time.Minute)
	fired := timer.Stop()
	c.Assert(fired, gc.Equals, true, gc.Commentf("expected Stop() to return true when the timer has already fired"))

	// Dequeue the channel; the notification should have already been
	// written to it when the timer expired.
	select {
	case <-timer.C():
	case <-time.After(3 * time.Second):
		c.Error("timeout waiting for timer to publish notification to its channel")
	}

	// Reset the timer, stop it again and check the reported hasFired status.
	timer.Reset(5 * time.Minute)
	fired = timer.Stop()
	c.Assert(fired, gc.Equals, false, gc.Commentf("expected Stop() to return false when the timer has not yet fired"))
}

func (fakeClockSuite) TestWaitAdvance(c *gc.C) {
	clk := NewFakeClock(time.Now())

	advancedCh := make(chan struct{})
	timeoutCh := clk.After(10 * time.Minute)
	timer := clk.NewTimer(10 * time.Minute)

	go func() {
		clk.WaitAdvance(2, 10*time.Minute)
		close(advancedCh)
	}()

	// Grab the timers channel so that the clock can detect that two
	// consumers are waiting on it.
	timerCh := timer.C()

	// Since both channels (timeout and timer) have been read by us,
	// the WaitAdvance call in the goroutine should unblock.
	select {
	case <-advancedCh:
	case <-time.After(3 * time.Second):
		c.Error("timeout waiting for WaitAdvance to return")
	}

	// A notification should have been enqueued to both channels
	select {
	case <-timeoutCh:
	case <-time.After(3 * time.Second):
		c.Error("timeout waiting for notification on clock.After() result")
	}
	select {
	case <-timerCh:
	case <-time.After(3 * time.Second):
		c.Error("timeout waiting for notification on clock.NewTimer().C() result")
	}
}
