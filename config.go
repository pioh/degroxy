package main

import (
	"flag"
	"net/url"
)

type config struct {
	listenAddress string
	targetURL     url.URL
}

func parseConfig() config {
	cfg := config{}
	var targetURL urlFlag

	flag.StringVar(&cfg.listenAddress, "l", ":8080", "Listen address")
	flag.Var(&targetURL, "t", "Proxy target url")

	flag.Parse()

	cfg.targetURL = url.URL(targetURL)

	return cfg
}

type urlFlag url.URL

func (v *urlFlag) String() string {
	if v != nil {
		u := url.URL(*v)
		return u.String()
	}
	return ""
}

func (v *urlFlag) Set(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return err
	}
	*v = urlFlag(*u)
	return nil
}
