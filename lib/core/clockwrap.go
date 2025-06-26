package core

import (
	"sync"
	"time"

	"github.com/keybase/clockwork"
)

// ClockWrap works around inherent race conditions in the Clockwork
// paradigm. The issue is this interleaving:
//
//	Go Routine 1: compute sleep duration
//	Go Routine 2: clock.Advance()
//	Go Routine 1: After(duration)
//
// Now there is no way to wake up GoRoutine 1. If only there was a way to check
// the time right before the After call in Go Routine 1, but the library
// doesn't allow for this. We need a different paradigm.
type ClockWrap struct {
	sync.Mutex
	cl clockwork.Clock
	st ClockWrapState
}

type ClockWrapState struct {
	i int
}

func NewClockWrap(cl clockwork.Clock) *ClockWrap {
	return &ClockWrap{
		cl: cl,
	}
}

func (c *ClockWrap) At(t time.Time) <-chan time.Time {
	c.Lock()
	defer c.Unlock()
	now := c.cl.Now()
	diff := t.Sub(now)
	if diff <= 0 {
		ret := make(chan time.Time, 1)
		ret <- now
		return ret
	}
	return c.cl.After(diff)
}

func (c *ClockWrap) PushTo(t time.Time, st *ClockWrapState) error {
	c.Lock()
	defer c.Unlock()

	if c.st.i != st.i {
		return InternalError("ClockWrap.Advance called with stale state; fighting go-routines?")
	}

	fake, ok := c.cl.(clockwork.FakeClock)
	if !ok {
		return InternalError("ClockWrap.Advance can only be used with FakeClock")
	}
	now := fake.Now()
	c.st.i++
	diff := t.Sub(now)
	if diff <= 0 {
		return nil
	}
	fake.Advance(diff)
	return nil
}
