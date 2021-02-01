package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/valyala/fasthttp"
)

func main() {
	wg := sync.WaitGroup{}

	for i := 7000; i < 10000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := proxy(
				fmt.Sprintf(":%d", i),
				fmt.Sprintf("localhost:%d", i+1),
			); err != nil {
				log.Println(err)
			}
		}(i)
	}

	wg.Wait()

	//cfg := parseConfig()
	//proxy := httputil.NewSingleHostReverseProxy(&cfg.targetURL)
	//limiter := newLimiter(cfg, proxy)
	//
	//http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
	//	limiter.handle(res, req)
	//})
	//
	//if err := http.ListenAndServe(cfg.listenAddress, nil); err != nil {
	//	log.Fatalf("Failed to listen and serve %s: %v\n", cfg.listenAddress, err)
	//}
}

func proxy(listen string, target string) error {
	cli := fasthttp.HostClient{
		Addr:                          target,
		DisableHeaderNamesNormalizing: true,
		DisablePathNormalizing:        true,
	}
	srv := fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			ctx.Request.SetHost(cli.Addr)
			if err := cli.Do(&ctx.Request, &ctx.Response); err != nil {
				ctx.Response.SetStatusCode(http.StatusInternalServerError)
				ctx.Response.SetBodyString(err.Error())
			}
		},
		DisablePreParseMultipartForm:  true,
		DisableHeaderNamesNormalizing: true,
	}

	return srv.ListenAndServe(listen)
}
