package rate

import (
	"fmt"
	"math"
	"sync/atomic"
	"testing"
	"time"
)

//func TestRate(t *testing.T) {
//	var threads int32 = 10
//	counter := NewCounter(time.Millisecond*10, time.Second)
//
//	done := make(chan bool)
//	wg := sync.WaitGroup{}
//	for thread := int32(0); thread < threads; thread++ {
//		wg.Add(1)
//		go func(thread int32) {
//			defer wg.Done()
//			ticker := time.NewTicker(time.Millisecond * 10)
//			defer ticker.Stop()
//			for {
//				select {
//				case <-done:
//					return
//				case <-ticker.C:
//					counter.Add(1)
//				}
//			}
//		}(thread)
//	}
//	dt := time.Millisecond * 100
//	sum1 := float64(0)
//	sum2 := float64(0)
//	sum3 := float64(0)
//	sum4 := float64(0)
//	time.Sleep(time.Second)
//	ticker := time.NewTicker(dt)
//	defer ticker.Stop()
//	for i := 0; i < int(time.Second*4/dt); i++ {
//		<-ticker.C
//		r1 := counter.Rate(dt, 0, dt)
//		sum1 += r1
//		r2 := counter.Rate(dt, dt, dt)
//		sum2 += r2
//		r3 := counter.Rate(dt*2, 0, dt)
//		sum3 += r3
//		sum4 += counter.Rate(dt*2, dt/3, dt*3)
//		t.Logf("%6.2f %6.2f %6.2f", r1, r2, r3)
//	}
//	close(done)
//	wg.Wait()
//
//	if math.Abs(sum1-4000.0) > 0.4 {
//		t.Fatalf("wrong counts: %f != %f", sum1, 4000.0)
//	}
//	if math.Abs(sum2-4000.0) > 0.4 {
//		t.Fatalf("wrong counts: %f != %f", sum2, 4000.0)
//	}
//	if math.Abs(sum3-4000.0) > 0.4 {
//		t.Fatalf("wrong counts: %f != %f", sum3, 4000.0)
//	}
//	if math.Abs(sum4-12000.0) > 0.4 {
//		t.Fatalf("wrong counts: %f != %f", sum4, 12000.0)
//	}
//}

func TestRate2(t *testing.T) {
	t.Parallel()
	tests := []struct {
		size     int64 // ms
		dt       int64
		tick     int64
		threads  int
		r1       float32
		r2       float32
		r3       float32
		wait     int64
		repeat   int
		interval int64
		offset   int64
		by       int64
		expect   float64
		dx       float64
	}{
		{10, 1, 1, 1, 1, 0, 0, 2, 4, 1, 0, 1, 1, 0},
		{100, 1, 1, 10, 1, 0, 0, 2, 4, 1, 0, 1, 10, 0},
		{100, 10, 1, 10, 1, 0, 0, 30, 40, 10, 0, 10, 100, 0},
		{100, 10, 1, 10, 1, 0, 0, 60, 40, 30, 10, 1, 10, 0},
	}
	for i, tt := range tests {
		tt := tt
		i := i
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			now := int64(100000)
			c := NewCounter(time.Duration(tt.dt), time.Duration(tt.size), func() int64 {
				return atomic.LoadInt64(&now)
			})
			next := make(chan bool)
			nextOk := make(chan bool)
			for k := 0; k < tt.threads; k++ {
				go func() {
					for {
						if exit := <-next; exit {
							nextOk <- true
							return
						}
						c.Add(int32(tt.r1))
						nextOk <- false
					}
				}()
			}
			w := tt.tick
			r := 0
			for {
				for k := 0; k < tt.threads; k++ {
					next <- false
				}
				for k := 0; k < tt.threads; k++ {
					<-nextOk
				}
				tt.r1 += tt.r2
				tt.r2 += tt.r3

				if w >= tt.wait {
					rate := c.Rate(time.Duration(tt.interval), time.Duration(tt.offset), time.Duration(tt.by))
					if math.Abs(tt.expect-rate) > tt.dx {
						t.Errorf("test: %v; r: %v; expect: %6.2f; got: %6.2f", i, r, tt.expect, rate)
					}
					r++
					if r >= tt.repeat {
						break
					}
					w = 0
				}

				w += tt.tick
				atomic.AddInt64(&now, tt.tick)
			}
			for k := 0; k < tt.threads; k++ {
				next <- true
			}
			for k := 0; k < tt.threads; k++ {
				<-nextOk
			}
		})
	}
}
