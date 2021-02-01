package main

import (
	"context"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/paulbellamy/ratecounter"
)

var qpsSent = ratecounter.NewRateCounter(1 * time.Second)
var qpsOK = ratecounter.NewRateCounter(1 * time.Second)
var qpsERR = ratecounter.NewRateCounter(1 * time.Second)
var latency = ratecounter.NewAvgRateCounter(time.Second)
var max int32

func main() {
	var queue int32
	for i := 0; i < 10; i++ {
		go func() {
			for {
				go func() {
					atomic.AddInt32(&queue, 1)
					qpsSent.Incr(1)
					test()
					atomic.AddInt32(&queue, -1)
				}()
				time.Sleep(time.Millisecond * 3)
			}
		}()
	}
	tick := time.Tick(time.Second)
	for {
		<-tick
		log.Printf("sent rps: %8d; ok: %8d; err: %4d; queue: %8d; latency avg: %6.0fms max: %6dms",
			qpsSent.Rate(), qpsOK.Rate(), qpsERR.Rate(), atomic.LoadInt32(&queue), latency.Rate(), atomic.LoadInt32(&max))
		atomic.StoreInt32(&max, 0)
	}
}
func test() {
	//1 1 0
	//1 1 1
	//1 2 3
	//1 4 7
	//1 8 15
	//1 16 31
	//-1 8 23
	//-1 4 19
	//1 2 21
	//1 1 22
	//1 0.5 22.5
	//1 1 23.5
	//-1 0.5 23
	//-1 0.25 22.75
	//
	//2 2 2
	//-1/2
	//
	//
	//
	//	good:
	//		x3 = x2 + (x2-x1)*step
	//	bad: step/2
	//		x3 = (x2+x1)/2

	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/", nil)
	var res *http.Response
	if err == nil {
		res, err = http.DefaultClient.Do(req)
		if err == nil {
			res.Body.Close()
		}
	}
	lat := time.Since(startTime).Milliseconds()
	latency.Incr(lat)
	if atomic.LoadInt32(&max) < int32(lat) {
		atomic.StoreInt32(&max, int32(lat))
	}

	if err != nil || res.StatusCode != 200 {
		qpsERR.Incr(1)
	} else {
		qpsOK.Incr(1)
	}
}
