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
	tick := time.Tick(time.Second)
	for {
		<-tick
		log.Printf("sent rps: %8d; ok: %8d; err: %4d; queue: %8d; latency avg: %6.0fms max: %6dms",
			qpsSent.Rate(), qpsOK.Rate(), qpsERR.Rate(), atomic.LoadInt32(&queue), latency.Rate(), atomic.LoadInt32(&max))
		atomic.StoreInt32(&max, 0)
	}
}
func test() {
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
