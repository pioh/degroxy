package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"sync"
	"sync/atomic"
	"time"

	"github.com/paulbellamy/ratecounter"
)

type Job struct {
	res http.ResponseWriter
	req *http.Request
}

type Snapshot struct {
	qps     int64
	threads int

	qpsIncome int64

	qps1i  int64
	qps10i int64
	qps60i int64

	qps1  int64
	qps10 int64
	qps60 int64

	qps1o  int64
	qps10o int64
	qps60o int64

	lat1  float64
	lat10 float64
	lat60 float64

	queue int64
}

func (s Snapshot) print() {
	log.Printf("qps: %v; threads: %v; income: %v; queue: %v\n", s.qps/60, s.threads, s.qpsIncome/60, s.queue)
	log.Printf("%8d\t%8d\t%8d\n", s.qps1i/60, s.qps10i/60, s.qps60i/60)
	log.Printf("%8d\t%8d\t%8d\n", s.qps1/60, s.qps10/60, s.qps60/60)
	log.Printf("%8d\t%8d\t%8d\n", s.qps1o/60, s.qps10o/60, s.qps60o/60)
	log.Printf("%8.1f\t%8.1f\t%8.1f\n", s.lat1/1000000, s.lat10/1000000, s.lat60/1000000)
}

type Limiter struct {
	m     *sync.Mutex
	proxy *httputil.ReverseProxy
	cfg   config

	qps     float64
	threads int

	threadsSem chan int

	history []Snapshot

	qpsIncome *ratecounter.RateCounter

	qps1i  *ratecounter.RateCounter
	qps10i *ratecounter.RateCounter
	qps60i *ratecounter.RateCounter

	qps1  *ratecounter.RateCounter
	qps10 *ratecounter.RateCounter
	qps60 *ratecounter.RateCounter

	qps1o  *ratecounter.RateCounter
	qps10o *ratecounter.RateCounter
	qps60o *ratecounter.RateCounter

	lat1  *ratecounter.AvgRateCounter
	lat10 *ratecounter.AvgRateCounter
	lat60 *ratecounter.AvgRateCounter

	queue    int64
	queueMax int64
}

func newLimiter(cfg config, proxy *httputil.ReverseProxy) *Limiter {
	l := &Limiter{
		m:     &sync.Mutex{},
		proxy: proxy,
		cfg:   cfg,

		//queue:      queue.New(1024),
		queueMax:   10,
		threadsSem: make(chan int, 1024),
		qps:        60 * 60,

		qpsIncome: ratecounter.NewRateCounter(time.Second),

		qps1i:  ratecounter.NewRateCounter(time.Second),
		qps10i: ratecounter.NewRateCounter(time.Second * 10),
		qps60i: ratecounter.NewRateCounter(time.Second * 60),

		qps1:  ratecounter.NewRateCounter(time.Second),
		qps10: ratecounter.NewRateCounter(time.Second * 10),
		qps60: ratecounter.NewRateCounter(time.Second * 60),

		qps1o:  ratecounter.NewRateCounter(time.Second),
		qps10o: ratecounter.NewRateCounter(time.Second * 10),
		qps60o: ratecounter.NewRateCounter(time.Second * 60),

		lat1:  ratecounter.NewAvgRateCounter(time.Second),
		lat10: ratecounter.NewAvgRateCounter(time.Second * 10),
		lat60: ratecounter.NewAvgRateCounter(time.Second * 60),

		threads: 9,
	}
	for i := 0; i < l.threads; i++ {
		l.threadsSem <- 0
	}

	go func() {
		ticker := time.Tick(time.Second * 1)
		for {
			<-ticker
			l.m.Lock()
			l.snapper()
			l.m.Unlock()
		}
	}()
	go func() {
		ticker := time.Tick(time.Second * 2)
		for {
			<-ticker
			l.m.Lock()
			l.teacher()
			l.m.Unlock()
		}
	}()
	return l
}

func (l *Limiter) snapper() {
	snap := l.snapshot()
	if len(l.history) < 60 {
		l.history = append(l.history, snap)
	} else {
		for i := 1; i < len(l.history); i++ {
			l.history[i-1] = l.history[i]
		}
		l.history[len(l.history)-1] = snap
	}
	snap.print()
}
func (l *Limiter) teacher() {
	//DL := 0
	h := l.history
	L := len(h)
	if L < 2 {
		return
	}
	//S := L - 10
	snap := h[L-1]

	//for i := S + 1; i < L; i++ {
	//	if h[i].queue > h[i-1].queue {
	//		DL++
	//	}
	//}
	queueMax := float64(snap.qps1i)*snap.lat1/1000/1000000/60 + float64(snap.threads)
	log.Println(queueMax)
	if float64(snap.queue) > queueMax {
		l.qps /= 1.01
	} else if float64(snap.queue) < queueMax {
		l.qps *= 1.01
	}
}

func (l *Limiter) snapshot() Snapshot {
	return Snapshot{
		qps:     int64(l.qps),
		threads: l.threads,

		qpsIncome: l.qpsIncome.Rate(),

		qps1i:  l.qps1i.Rate(),
		qps10i: l.qps10i.Rate(),
		qps60i: l.qps60i.Rate(),

		qps1:  l.qps1.Rate(),
		qps10: l.qps10.Rate(),
		qps60: l.qps60.Rate(),

		qps1o:  l.qps1o.Rate(),
		qps10o: l.qps10o.Rate(),
		qps60o: l.qps60o.Rate(),

		lat1:  l.lat1.Rate(),
		lat10: l.lat10.Rate(),
		lat60: l.lat60.Rate(),

		queue: atomic.LoadInt64(&l.queue),
	}
}
func (l *Limiter) runJob(job *Job) {
	<-l.threadsSem
	defer func() {
		l.threadsSem <- 0
	}()

	l.qps1.Incr(60)
	l.qps10.Incr(6)
	l.qps60.Incr(1)

	job.req.Host = l.cfg.targetURL.Host

	startTime := time.Now()
	l.proxy.ServeHTTP(job.res, job.req)
	lat := time.Since(startTime)

	l.lat1.Incr(lat.Nanoseconds())
	l.lat10.Incr(lat.Nanoseconds())
	l.lat60.Incr(lat.Nanoseconds())

	l.qps1o.Incr(60)
	l.qps10o.Incr(6)
	l.qps60o.Incr(1)
}

func (l *Limiter) handle(res http.ResponseWriter, req *http.Request) {
	l.qpsIncome.Incr(60)
	l.m.Lock()
	s := l.snapshot()
	atomic.AddInt64(&l.queue, 1)
	defer atomic.AddInt64(&l.queue, -1)
	l.m.Unlock()

	if l.qps1i.Rate() > s.qps {
		http.Error(res, "service is temporarily overloaded", http.StatusServiceUnavailable)
		return
	}
	l.qps1i.Incr(60)
	l.qps10i.Incr(6)
	l.qps60i.Incr(1)

	l.runJob(&Job{
		res: res,
		req: req,
	})
}
