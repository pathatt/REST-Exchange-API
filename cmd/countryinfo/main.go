package main

import (
	"log"
	"net/http"
	"os"
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

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	log.Println("listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))

}
