package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
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

type countryAlphaItem struct {
	Name struct {
		Common string `json:"common"`
	} `json:"name"`

	Continents []string          `json:"continents"`
	Population int               `json:"population"`
	Area       float64           `json:"area"`
	Languages  map[string]string `json:"languages"`
	Borders    []string          `json:"borders"`

	Flags struct {
		PNG string `json:"png"`
	} `json:"flags"`

	Capital []string `json:"capital"`

	Currencies map[string]struct {
		Symbol string `json:"symbol"`
		Name   string `json:"name"`
	} `json:"currencies"`
}

type countryAlphaResponse []countryAlphaItem

func decodeCountryAlpha(resp *http.Response) (countryAlphaItem, int, error) {
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return countryAlphaItem{}, http.StatusBadGateway, err
	}

	// Try array first: [{...}]
	var arr countryAlphaResponse
	if err := json.Unmarshal(raw, &arr); err == nil {
		if len(arr) == 0 {
			return countryAlphaItem{}, http.StatusNotFound, fmt.Errorf("no country data")
		}
		return arr[0], http.StatusOK, nil
	}

	// Fallback: {...}
	var obj countryAlphaItem
	if err := json.Unmarshal(raw, &obj); err == nil {
		// basic sanity check
		if obj.Name.Common == "" {
			return countryAlphaItem{}, http.StatusBadGateway, fmt.Errorf("unexpected country payload")
		}
		return obj, http.StatusOK, nil
	}

	return countryAlphaItem{}, http.StatusBadGateway, fmt.Errorf("could not decode country payload")
}

type InfoResponse struct {
	Name       string            `json:"name"`
	Continents []string          `json:"continents"`
	Population int               `json:"population"`
	Area       float64           `json:"area"`
	Languages  map[string]string `json:"languages"`
	Borders    []string          `json:"borders"`
	Flag       string            `json:"flag"`
	Capital    string            `json:"capital"`
}

func extractCode(path, prefix string) string {
	code := strings.TrimPrefix(path, prefix)
	return strings.ToLower(strings.Trim(code, "/"))
}

func fetchCountryInfo(r *http.Request, client *http.Client, twoLetter string) (InfoResponse, int, error) {
	twoLetter = strings.ToLower(strings.TrimSpace(twoLetter))
	if len(twoLetter) != 2 {
		return InfoResponse{}, http.StatusBadRequest, fmt.Errorf("country code must be 2 letters")
	}

	u, _ := url.Parse(countryBase + "/v3.1/alpha/" + twoLetter)
	q := u.Query()
	q.Set("fields", "name,continents,population,area,languages,borders,flags,capital")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, u.String(), nil)
	if err != nil {
		return InfoResponse{}, http.StatusInternalServerError, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return InfoResponse{}, http.StatusBadGateway, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return InfoResponse{}, resp.StatusCode, fmt.Errorf("countries api returned %d", resp.StatusCode)
	}

	c, st, err := decodeCountryAlpha(resp)
	if err != nil {
		return InfoResponse{}, st, err
	}

	capital := ""
	if len(c.Capital) > 0 {
		capital = c.Capital[0]
	}

	return InfoResponse{
		Name:       c.Name.Common,
		Continents: c.Continents,
		Population: c.Population,
		Area:       c.Area,
		Languages:  c.Languages,
		Borders:    c.Borders,
		Flag:       c.Flags.PNG,
		Capital:    capital,
	}, http.StatusOK, nil
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

		// Time in seconds since last re/start
		uptime := int64(time.Since(start).Seconds())

		// Check if both APIs are up
		countryStatus := probe(client, countryBase+"/v3.1/all")
		currencyStatus := probe(client, currencyBase+"/NOK")
		// and return status code (int) based on result
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
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		code := extractCode(r.URL.Path, "/countryinfo/v1/info/")
		info, status, err := fetchCountryInfo(r, client, code)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		writeJSON(w, http.StatusOK, info)
	})

	// EXCHANGE
	mux.HandleFunc("/countryinfo/v1/exchange/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "exchange endpoint (TODO) use /countryinfo/v1/exchange/{code}")
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
