package rtimer

import (
	"math/rand"
	"time"
)

type Timer interface {
	Reset(from time.Duration, to time.Duration)
	Chan() <-chan time.Time
}

type RandomTimer struct {
	r *rand.Rand
	t *time.Timer
}

func Default() *RandomTimer {
	return &RandomTimer{
		r: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (r *RandomTimer) Reset(from time.Duration, to time.Duration) {
	randomRange := int64(to - from)
	delay := from
	if randomRange == 0 {
		// no random part
	} else {
		delay = delay + (time.Duration(r.r.Int63n(randomRange)))
	}
	if r.t == nil {
		r.t = time.NewTimer(delay)
	} else {
		r.t.Stop()
		select {
		case <-r.t.C:
		default:
		}
		r.t.Reset(delay)
	}
}

func (r *RandomTimer) Chan() <-chan time.Time {
	if r.t == nil {
		return nil
	}
	return r.t.C
}
