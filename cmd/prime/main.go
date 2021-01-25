package main

import (
	"crypto/rand"
	"log"
	"math/big"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		for {
			num, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 1<<10))
			if err != nil {
				http.Error(res, err.Error(), 500)
				return
			}
			if num.ProbablyPrime(20) {
				s := num.String()
				if _, err := res.Write([]byte(s)); err != nil {
					log.Printf("Failed write response: %v", err)
				}
				log.Println(s)
				return
			}
		}
	})

	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}
