package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	countryBase  = "http://129.241.150.113:8080"
	currencyBase = "http://129.241.150.113:9090/currency"
	apiVersion   = "v1"
)

// Route paths:
const (
	basePath     = "/countryinfo/" + apiVersion
	statusPath   = basePath + "/status/"
	infoPath     = basePath + "/info/"
	exchangePath = basePath + "/exchange/"
)

// My API response models:
type StatusResponse struct {
	CountriesAPI  any    `json:"restcountriesapi"`
	CurrenciesAPI any    `json:"currenciesapi"`
	Version       string `json:"version"`
	Uptime        int64  `json:"uptime"`
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

// RestCountries API models:
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

type ExchangeRateEntry map[string]float64

type ExchangeResponse struct {
	Country       string              `json:"country"`
	BaseCurrency  string              `json:"base-currency"`
	ExchangeRates []ExchangeRateEntry `json:"exchange-rates"`
}

// HTTP helpers:

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

func extractCode(path, prefix string) string {
	code := strings.TrimPrefix(path, prefix)
	return strings.ToLower(strings.Trim(code, "/"))
}

// Countries API decoding and fetch helpers:
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
		if obj.Name.Common == "" {
			return countryAlphaItem{}, http.StatusBadGateway, fmt.Errorf("unexpected country payload")
		}
		return obj, http.StatusOK, nil
	}

	return countryAlphaItem{}, http.StatusBadGateway, fmt.Errorf("could not decode country payload")
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

type currencyRatesResponse struct {
	Result   string             `json:"result"`
	BaseCode string             `json:"base_code"`
	Rates    map[string]float64 `json:"rates"`
}

func fetchCurrencyRates(r *http.Request, client *http.Client, baseCurrency string) (map[string]float64, int, error) {
	baseCurrency = strings.ToUpper(strings.TrimSpace(baseCurrency))
	if len(baseCurrency) != 3 {
		return nil, http.StatusBadRequest, fmt.Errorf("base currency must be 3 letters")
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, currencyBase+"/"+baseCurrency, nil)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("currencies api returned %d", resp.StatusCode)
	}

	var payload currencyRatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, http.StatusBadGateway, err
	}
	if strings.ToLower(payload.Result) != "success" || payload.BaseCode == "" {
		return nil, http.StatusBadGateway, fmt.Errorf("unexpected currencies payload")
	}

	return payload.Rates, http.StatusOK, nil
}

func pickDeterministicCurrencyCode(m map[string]struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}) string {
	if len(m) == 0 {
		return ""
	}

	codes := make([]string, 0, len(m))
	for code := range m {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes[0]
}

type CountryCore struct {
	Name         string
	Borders      []string
	BaseCurrency string
}

func fetchCountryCore(r *http.Request, client *http.Client, twoLetter string) (CountryCore, int, error) {
	twoLetter = strings.ToLower(strings.TrimSpace(twoLetter))
	if len(twoLetter) != 2 {
		return CountryCore{}, http.StatusBadRequest, fmt.Errorf("country code must be 2 letters")
	}

	u, _ := url.Parse(countryBase + "/v3.1/alpha/" + twoLetter)
	q := u.Query()
	q.Set("fields", "name,borders,currencies")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, u.String(), nil)
	if err != nil {
		return CountryCore{}, http.StatusInternalServerError, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return CountryCore{}, http.StatusBadGateway, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return CountryCore{}, resp.StatusCode, fmt.Errorf("countries api returned %d", resp.StatusCode)
	}

	c, st, err := decodeCountryAlpha(resp)
	if err != nil {
		return CountryCore{}, st, err
	}

	baseCur := pickDeterministicCurrencyCode(c.Currencies)
	if baseCur == "" {
		return CountryCore{}, http.StatusBadGateway, fmt.Errorf("missing currency for country")
	}

	return CountryCore{
		Name:         c.Name.Common,
		Borders:      c.Borders,
		BaseCurrency: baseCur,
	}, http.StatusOK, nil
}

type neighbourItem struct {
	CCA3       string `json:"cca3"`
	Currencies map[string]struct {
		Symbol string `json:"symbol"`
		Name   string `json:"name"`
	} `json:"currencies"`
}
type neighbourResponse []neighbourItem

func fetchNeighbourCurrenciesByCCA3(r *http.Request, client *http.Client, cca3Codes []string) (map[string]string, int, error) {
	// returns map[CCA3]CurrencyCode
	clean := make([]string, 0, len(cca3Codes))
	seen := map[string]bool{}

	for _, c := range cca3Codes {
		c = strings.ToUpper(strings.TrimSpace(c))
		if len(c) != 3 || seen[c] {
			continue
		}
		seen[c] = true
		clean = append(clean, c)
	}

	if len(clean) == 0 {
		return map[string]string{}, http.StatusOK, nil
	}

	u, _ := url.Parse(countryBase + "/v3.1/alpha")
	q := u.Query()
	q.Set("codes", strings.Join(clean, ","))
	q.Set("fields", "cca3,currencies")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("countries api returned %d", resp.StatusCode)
	}

	var payload neighbourResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, http.StatusBadGateway, err
	}

	out := make(map[string]string, len(payload))
	for _, n := range payload {
		out[n.CCA3] = pickDeterministicCurrencyCode(n.Currencies)
	}
	return out, http.StatusOK, nil
}

func main() {
	start := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}

	mux := http.NewServeMux()

	// STATUS
	mux.HandleFunc(statusPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		uptime := int64(time.Since(start).Seconds())

		countryStatus := probe(client, countryBase+"/v3.1/all")
		currencyStatus := probe(client, currencyBase+"/NOK")

		httpStatus := http.StatusOK
		if countryStatus != http.StatusOK || currencyStatus != http.StatusOK {
			httpStatus = http.StatusBadGateway
		}

		writeJSON(w, httpStatus, StatusResponse{
			CountriesAPI:  countryStatus,
			CurrenciesAPI: currencyStatus,
			Version:       apiVersion,
			Uptime:        uptime,
		})
	})

	// INFO
	mux.HandleFunc(infoPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		code := extractCode(r.URL.Path, infoPath)
		info, status, err := fetchCountryInfo(r, client, code)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		writeJSON(w, http.StatusOK, info)
	})

	// EXCHANGE
	mux.HandleFunc(exchangePath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		code := extractCode(r.URL.Path, exchangePath)

		// 1) base country info (name, borders, base currency)
		core, status, err := fetchCountryCore(r, client, code)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		// If no borders: return empty list (still 200)
		if len(core.Borders) == 0 {
			writeJSON(w, http.StatusOK, ExchangeResponse{
				Country:       core.Name,
				BaseCurrency:  core.BaseCurrency,
				ExchangeRates: []ExchangeRateEntry{},
			})
			return
		}

		// 2) neighbour currency codes (batch)
		neighCurByCCA3, st2, err := fetchNeighbourCurrenciesByCCA3(r, client, core.Borders)
		if err != nil {
			http.Error(w, err.Error(), st2)
			return
		}

		// 3) all rates from base currency (one call)
		rates, st3, err := fetchCurrencyRates(r, client, core.BaseCurrency)
		if err != nil {
			http.Error(w, err.Error(), st3)
			return
		}

		// 4) build response entries in border order, de-duplicating currencies
		seenCur := map[string]bool{}
		exchangeRates := make([]ExchangeRateEntry, 0, len(core.Borders))

		for _, cca3 := range core.Borders {
			cca3 = strings.ToUpper(cca3)
			cur := neighCurByCCA3[cca3]
			if cur == "" || seenCur[cur] {
				continue
			}
			seenCur[cur] = true

			rate, ok := rates[cur]
			if !ok {
				// currency API may not have the code; skip gracefully
				continue
			}
			exchangeRates = append(exchangeRates, ExchangeRateEntry{cur: rate})
		}

		writeJSON(w, http.StatusOK, ExchangeResponse{
			Country:       core.Name,
			BaseCurrency:  core.BaseCurrency,
			ExchangeRates: exchangeRates,
		})
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
