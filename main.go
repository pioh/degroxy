package main

import (
	"log"
	"net/http"
	"net/http/httputil"
)

func main() {

	cfg := parseConfig()
	proxy := httputil.NewSingleHostReverseProxy(&cfg.targetURL)
	limiter := newLimiter(cfg, proxy)

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		limiter.handle(res, req)
	})

	if err := http.ListenAndServe(cfg.listenAddress, nil); err != nil {
		log.Fatalf("Failed to listen and serve %s: %v\n", cfg.listenAddress, err)
	}
}
