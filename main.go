package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/countryinfo/v1/status/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "status endpoint (TODO)")
	})
	mux.HandleFunc("/countryinfo/v1/info/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "info endpoint (TODO) use /countryinfo/v1/info/{code}")
	})
	mux.HandleFunc("/countryinfo/v1/exchange/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "exchange endpoint (TODO) use /countryinfo/v1/exchange/{code}")
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
