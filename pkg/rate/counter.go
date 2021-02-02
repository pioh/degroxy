package rate

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kpango/fastime"
)

type Counter struct {
	counter           []int32
	dt                int64
	lastFillZerosTime int64

	sync.Mutex

	now func() int64
}

func NewCounter(dt time.Duration, size time.Duration) *Counter {
	now := fastime.UnixNanoNow
	c := &Counter{
		counter:           make([]int32, int64(size/dt)+1),
		dt:                dt.Nanoseconds(),
		lastFillZerosTime: now(),
		now:               now,
	}
	return c
}

func (c *Counter) i(t int64) int {
	return int((t / c.dt) % int64(len(c.counter)))
}

func (c *Counter) shift(i int, shift int) int {
	i += shift
	l := len(c.counter)
	if i < 0 {
		i += l
	}
	if i >= l {
		i -= l
	}
	return i
}

func (c *Counter) Add(count int32) {
	now := c.now()
	i := c.fillZeros(now)
	atomic.AddInt32(&c.counter[i], count)
}

// returns now
func (c *Counter) fillZeros(now int64) int {
	last := atomic.LoadInt64(&c.lastFillZerosTime)
	n := c.i(now)
	l := c.i(last)
	if n == l && now-last <= c.dt {
		return n
	}

	c.Lock()
	defer c.Unlock()
	last = atomic.LoadInt64(&c.lastFillZerosTime)
	l = c.i(last)

	//log.Println(n,l, now-last>c.dt)
	for n != l || now-last > c.dt {
		last += c.dt
		//cc := atomic.LoadInt32(&c.counter[l])
		l = c.shift(l, 1)
		//log.Println(l, cc, atomic.LoadInt32(&c.counter[l]))
		atomic.StoreInt32(&c.counter[l], 0)
	}
	atomic.StoreInt64(&c.lastFillZerosTime, now)
	return n
}

var ErrTimeOutOfRange = errors.New("time interval out of range")

func interpolation3(x1, x2, x3, y1, y2, y3, x float64) (y float64) {
	y = y1*(((x-x2)*(x-x3))/((x1-x2)*(x1-x3))) +
		y2*(((x-x1)*(x-x3))/((x2-x1)*(x2-x3))) +
		y3*(((x-x1)*(x-x2))/((x3-x1)*(x3-x2)))
	return y
}
func interpolation2(x1, x2, y1, y2, x float64) (y float64) {
	y = y2 + (y1-y2)*((x-x2)/(x1-x2))
	return y
}

func (c *Counter) xy(t int64, now int64) (x float64, y float64) {
	left := t / c.dt * c.dt
	endLeft := now / c.dt * c.dt
	//beginLeft := (now - c.dt*int64(len(c.counter))) / c.dt * c.dt
	y = float64(atomic.LoadInt32(&c.counter[c.i(t)]))
	if left != endLeft {
		x = float64(left + c.dt/2)
	} else if now-left < c.dt {
		return 0, 0
	} else {
		x = float64(left/2 + now/2)
		y *= float64(c.dt) / float64(now-left)
	}
	return x, y
}

func (c *Counter) interpolate(t int64, now int64, right bool) (y float64) {
	left := t / c.dt * c.dt
	endLeft := now / c.dt * c.dt
	beginLeft := (now - c.dt*int64(len(c.counter))) / c.dt * c.dt
	points := 2
	var x1, x2, x3, y1, y2, y3, x float64
	x1, y1 = c.xy(left, now)
	if left == beginLeft {
		x2, y2 = c.xy(left+c.dt, now)
	} else if left == endLeft {
		x2, y2 = c.xy(left-c.dt, now)
	} else {
		x2, y2 = c.xy(left+c.dt, now)
		x3, y3 = c.xy(left-c.dt, now)
		points = 3
	}
	if points == 3 && x3 == 0 {
		points--
	}
	if x2 == 0 {
		x2, y2 = x3, y3
		points--
	}
	if x1 == 0 {
		x1, y1 = x2, y2
		points--
	}
	x = float64(t/2 + left/2)
	if points == 3 {
		y = interpolation3(x1, x2, x3, y1, y2, y3, x)
	} else if points == 2 {
		y = interpolation2(x1, x2, y1, y2, x)
	} else {
		y = y1
	}
	if right {
		y *= float64(t-left) / float64(c.dt)
	} else {
		y *= float64(left+c.dt-t) / float64(c.dt)
	}
	return y
}

// Sum counters for interval (2sec for example) with offset (10sec)
// max interval = keepHistory
// max offset = keepHistory - interval
func (c *Counter) Sum(interval time.Duration, offset time.Duration) float64 {
	if offset < 0 || interval < 0 || int64(offset+interval) > c.dt*int64(len(c.counter)) {
		panic(ErrTimeOutOfRange)
	}
	if interval == 0 {
		return 0
	}
	now := c.now()
	c.fillZeros(now)

	from := now - offset.Nanoseconds()
	to := from - interval.Nanoseconds()

	sum := float64(0)

	i := c.i(from - c.dt)
	for t := from - c.dt; t >= to+c.dt; t -= c.dt { // грузим все целые ячейки (кроме первой и последней)
		sum += float64(atomic.LoadInt32(&c.counter[i]))
		i = c.shift(i, -1)
	}
	if from-to >= c.dt {
		sum += c.interpolate(from, now, true)
		sum += c.interpolate(to, now, false)
	} else {
		sum += c.interpolate(from, now, true)
		sum -= c.interpolate(to, now, true)
	}

	return sum
}

func (c *Counter) Rate(interval time.Duration, offset time.Duration, dt time.Duration) float64 {
	sum := c.Sum(interval, offset)
	return sum * (float64(dt) / float64(interval))
}
