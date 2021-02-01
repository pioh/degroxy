package main

import (
	"log"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	srv := fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			//logger.Info("req", zap.ByteString("path: ", ctx.Path()))
			ctx.Response.SetBody(ctx.Path())
		},
		DisablePreParseMultipartForm:  true,
		DisableHeaderNamesNormalizing: true,
	}

	if err := srv.ListenAndServe(":10000"); err != nil {
		log.Println(err)
	}

	//http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
	//	for {
	//		num, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 1<<8))
	//		if err != nil {
	//			http.Error(res, err.Error(), 500)
	//			return
	//		}
	//		if num.ProbablyPrime(20) {
	//			s := num.String()
	//			if _, err := res.Write([]byte(s)); err != nil {
	//				log.Printf("Failed write response: %v", err)
	//			}
	//			log.Println(s)
	//			return
	//		}
	//	}
	//})
	//
	//if err := http.ListenAndServe(":8081", nil); err != nil {
	//	log.Fatal(err)
	//}
}
