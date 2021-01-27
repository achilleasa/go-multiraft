package clock

import "time"

// Clock defines an API for accessing the current time and for implementing
// timeouts / timers.
type Clock interface {
	// Now returns the current time.
	Now() time.Time

	// After waits for the duration to elapse and then sends the current time on
	// the returned channel.
	After(time.Duration) <-chan time.Time

	// NewTimer creates a new Timer that will send the current time on its
	// channel after at least duration d.
	NewTimer(time.Duration) Timer
}

// Timer defines an API for accessing a timer obtained via a clock instance.
type Timer interface {
	// C returns a channel where the timer will send the current time once it
	// expires.
	C() <-chan time.Time

	// Reset the timer to expire after at least duration d.
	Reset(time.Duration)

	// Stop the timer from firing. It returns true if the timer was active and
	// false otherwise. if Stop returns false, the caller must first drain the
	// channel channel before calling Reset().
	Stop() bool
}
