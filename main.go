package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	countryBase  = "http://129.241.150.113:8080"
	currencyBase = "http://129.241.150.113:9090/currency"
	apiVersion   = "v1"
)

type StatusResponse struct {
	CountriesAPI  any    `json:"restcountriesapi"`
	CurrenciesAPI any    `json:"currenciesapi"`
	Version       string `json:"version"`
	Uptime        int64  `json:"uptime"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func probe(client *http.Client, url string) int {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func main() {
	start := time.Now()

	client := &http.Client{Timeout: 10 * time.Second}

	mux := http.NewServeMux()

	// STATUS
	mux.HandleFunc("/countryinfo/v1/status/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Time since last re/start
		uptime := int64(time.Since(start).Seconds())

		// Check if both APIs are up
		countryStatus := probe(client, countryBase+"/v3.1/all")
		currencyStatus := probe(client, currencyBase+"/NOK")
		// and return status code based on result
		httpStatus := http.StatusOK
		if countryStatus != http.StatusOK || currencyStatus != http.StatusOK {
			httpStatus = http.StatusBadGateway
		}

		// Write to endpoint
		writeJSON(w, httpStatus, StatusResponse{
			CountriesAPI:  countryStatus,
			CurrenciesAPI: currencyStatus,
			Version:       apiVersion,
			Uptime:        uptime,
		})
	})

	// INFO
	mux.HandleFunc("/countryinfo/v1/info/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "info endpoint (TODO) use /countryinfo/v1/info/{code}")
	})

	// EXCHANGE
	mux.HandleFunc("/countryinfo/v1/exchange/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "exchange endpoint (TODO) use /countryinfo/v1/exchange/{code}")
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
