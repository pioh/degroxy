package rate

import (
	"math"
	"sync"
	"testing"
	"time"
)

func TestRate(t *testing.T) {
	var threads int32 = 10
	counter := NewCounter(time.Millisecond*10, time.Second)

	done := make(chan bool)
	wg := sync.WaitGroup{}
	for thread := int32(0); thread < threads; thread++ {
		wg.Add(1)
		go func(thread int32) {
			defer wg.Done()
			ticker := time.NewTicker(time.Millisecond * 10)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					counter.Add(1)
				}
			}
		}(thread)
	}
	dt := time.Millisecond * 100
	sum1 := float64(0)
	sum2 := float64(0)
	sum3 := float64(0)
	sum4 := float64(0)
	time.Sleep(time.Second)
	ticker := time.NewTicker(dt)
	defer ticker.Stop()
	for i := 0; i < int(time.Second*4/dt); i++ {
		<-ticker.C
		r1 := counter.Rate(dt, 0, dt)
		sum1 += r1
		r2 := counter.Rate(dt, dt, dt)
		sum2 += r2
		r3 := counter.Rate(dt*2, 0, dt)
		sum3 += r3
		sum4 += counter.Rate(dt*2, dt/3, dt*3)
		t.Logf("%6.2f %6.2f %6.2f", r1, r2, r3)
	}
	close(done)
	wg.Wait()

	if math.Abs(sum1-4000.0) > 0.4 {
		t.Fatalf("wrong counts: %f != %f", sum1, 4000.0)
	}
	if math.Abs(sum2-4000.0) > 0.4 {
		t.Fatalf("wrong counts: %f != %f", sum2, 4000.0)
	}
	if math.Abs(sum3-4000.0) > 0.4 {
		t.Fatalf("wrong counts: %f != %f", sum3, 4000.0)
	}
	if math.Abs(sum4-12000.0) > 0.4 {
		t.Fatalf("wrong counts: %f != %f", sum4, 12000.0)
	}
}
