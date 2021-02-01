package rate

import (
	"context"
	"sync/atomic"
	"time"
)

type Counter struct {
	counter []int32 // [time][thread]
	time    []int64 // [start time nanoseconds by counter index]

	intervals   int32
	threads     int32
	dt          int64
	keepHistory int64

	current *int32 // atomic use

	ticker *time.Ticker
}

func NewCounter(ctx context.Context, dt time.Duration, keepHistory time.Duration, threads int32) *Counter {
	intervals := int32(keepHistory/dt) + 4 // round up, and -2 +1 safe cell from now
	c := &Counter{
		keepHistory: keepHistory.Nanoseconds(),
		intervals:   intervals,
		threads:     threads,
		dt:          dt.Nanoseconds(),
		current:     new(int32),
		counter:     make([]int32, intervals*threads),
		time:        make([]int64, intervals),
		ticker:      time.NewTicker(dt),
	}
	now := time.Now().UnixNano()
	c.time[0] = now
	c.time[1] = now - dt.Nanoseconds()

	for i := int32(2); i < intervals; i++ {
		fromEnd := intervals - i // [_,_,... 3,2,1]
		c.time[i] = now - dt.Nanoseconds()*int64(fromEnd)
	}

	go func() {
		for {
			select {
			case now := <-c.ticker.C:
				c.tick(now)
			case <-ctx.Done():
				c.ticker.Stop()
			}
		}
	}()

	return c
}

func (c *Counter) tick(now time.Time) {
	current := atomic.LoadInt32(c.current)
	current++
	if current >= c.intervals {
		current = 0
	}
	atomic.StoreInt64(&c.time[current], now.UnixNano())
	for i := int32(0); i < c.threads; i++ {
		atomic.StoreInt32(&c.counter[current*c.threads+i], 0)
	}
	atomic.StoreInt32(c.current, current)
}

func (c *Counter) Add(count int32, thread int32) {
	current := atomic.LoadInt32(c.current)
	atomic.AddInt32(&c.counter[current*c.threads+thread], count)
}

// Sum counters for interval (2sec for example) with offset (10sec) + dt
// max interval = keepHistory
// max offset = keepHistory - interval
func (c *Counter) Sum(interval time.Duration, offset time.Duration) float64 {
	current := atomic.LoadInt32(c.current)
	now := time.Now().UnixNano()
	from := now - offset.Nanoseconds() - c.dt
	to := from - interval.Nanoseconds()
	current -= int32(offset.Nanoseconds()/c.dt) + 1

	sum := float64(0)

	for {
		prev := current + 1
		if current < 0 {
			current = c.intervals - 1
		}
		if prev < 0 {
			prev = c.intervals - 1
		}

		t := atomic.LoadInt64(&c.time[current])
		dt := atomic.LoadInt64(&c.time[prev]) - t

		if t > from {
			current--
			continue
		}
		s := float64(0)
		for i := int32(0); i < c.threads; i++ {
			s += float64(atomic.LoadInt32(&c.counter[current*c.threads+i]))
		}
		if t < to {
			s *= float64(t+dt-to) / float64(dt)
			sum += s
			break
		}

		if t+dt > from {
			s *= float64(from-t) / float64(dt)
		}
		sum += s

		current--
	}

	return sum
}

func (c *Counter) Rate(interval time.Duration, offset time.Duration, dt time.Duration) float64 {
	sum := c.Sum(interval, offset)
	return sum * (float64(dt) / float64(interval))
}
