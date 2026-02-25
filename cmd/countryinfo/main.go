package main

import (
	"log"
	"net/http"
	"time"

	"assignment-1/internal/api"
	"assignment-1/internal/clients"
)

func main() {
	httpClient := &http.Client{Timeout: 10 * time.Second}

	countries := &clients.CountriesClient{
		BaseURL: "http://129.241.150.113:8080",
		HTTP:    httpClient,
	}
	currency := &clients.CurrencyClient{
		BaseURL: "http://129.241.150.113:9090/currency",
		HTTP:    httpClient,
	}

	env := &api.HandlerEnv{
		Countries: countries,
		Currency:  currency,
		Start:     time.Now(),
	}

	mux := http.NewServeMux()
	env.Register(mux)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
