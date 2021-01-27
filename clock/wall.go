package clock

import "time"

// WallClock implements a clock using the time package from the Go standard library.
var WallClock Clock = wallClock{}

type wallClock struct{}

// Now returns the current time.
func (wallClock) Now() time.Time { return time.Now() }

// After waits for the duration to elapse and then sends the current time on
// the returned channel.
func (wallClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

// NewTimer creates a new Timer that will send the current time on its
// channel after at least duration d.
func (wallClock) NewTimer(d time.Duration) Timer {
	return wallClockTimer{time.NewTimer(d)}
}

type wallClockTimer struct {
	t *time.Timer
}

func (wt wallClockTimer) C() <-chan time.Time   { return wt.t.C }
func (wt wallClockTimer) Reset(d time.Duration) { _ = wt.t.Reset(d) }
func (wt wallClockTimer) Stop() bool            { return wt.t.Stop() }
