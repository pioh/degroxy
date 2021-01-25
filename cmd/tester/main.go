package main

import (
	"log"
	"net/http"
	"time"

	"github.com/paulbellamy/ratecounter"
)

var qpsSent = ratecounter.NewRateCounter(1 * time.Second)
var qpsOK = ratecounter.NewRateCounter(1 * time.Second)
var qpsERR = ratecounter.NewRateCounter(1 * time.Second)
var latency = ratecounter.NewAvgRateCounter(time.Second)

func main() {
	for i := 0; i < 64; i++ {
		go func() {
			for ; ; {
				qpsSent.Incr(1)
				go test()
				time.Sleep(time.Millisecond * 500)
			}
		}()
	}
	tick := time.Tick(time.Second)
	for {
		<-tick
		log.Printf("sent: %v;\t ok: %v;\t err: %v;\t latency: %.2fms", qpsSent.Rate(), qpsOK.Rate(), qpsERR.Rate(), latency.Rate())
	}
}
func test() {
	startTime := time.Now()

	res, err := http.Get("http://localhost:8081/")

	latency.Incr(time.Since(startTime).Milliseconds())

	if err != nil || res.StatusCode != 200 {
		qpsERR.Incr(1)
	} else {
		qpsOK.Incr(1)
	}
	if err == nil {
		res.Body.Close()
	}
}
