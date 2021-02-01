package rate

import (
	"context"
	"log"
	"sync"
	"testing"
	"time"
)

func TestRate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var threads int32 = 32
	counter := NewCounter(ctx, time.Millisecond*100, time.Minute*2, threads)

	done := make(chan bool)
	wg := sync.WaitGroup{}
	for thread := int32(0); thread < threads; thread++ {
		wg.Add(1)
		go func(thread int32) {
			defer wg.Done()
			ticker := time.NewTicker(time.Millisecond * 10 * time.Duration(thread + 1))
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					counter.Add(1, thread)
				}
			}
		}(thread)
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for i := 0; i < 10; i++ {
		<-ticker.C
		r1 := counter.Rate(time.Second, 0, time.Second)
		r2 := counter.Rate(time.Second, time.Second, time.Second)
		r3 := counter.Rate(time.Second*2, 0, time.Second)
		log.Printf("%4.2f %4.2f %4.2f", r1, r2, r3)
		time.Sleep(time.Second)
	}
	close(done)
	wg.Wait()
}
