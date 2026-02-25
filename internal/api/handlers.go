package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"assignment-1/internal/clients"
)

type HandlerEnv struct {
	Countries *clients.CountriesClient
	Currency  *clients.CurrencyClient
	Start     time.Time
}

func (e *HandlerEnv) Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	uptime := int64(time.Since(e.Start).Seconds())

	countryStatus := Probe(e.Countries.HTTP, e.Countries.BaseURL+"/v3.1/all")
	currencyStatus := Probe(e.Currency.HTTP, strings.TrimRight(e.Currency.BaseURL, "/")+"/NOK")

	httpStatus := http.StatusOK
	if countryStatus != http.StatusOK || currencyStatus != http.StatusOK {
		httpStatus = http.StatusBadGateway
	}

	WriteJSON(w, httpStatus, StatusResponse{
		CountriesAPI:  countryStatus,
		CurrenciesAPI: currencyStatus,
		Version:       APIVersion,
		Uptime:        uptime,
	})
}

func (e *HandlerEnv) Info(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	code := ExtractCode(r.URL.Path, InfoPath)
	if code == "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Use: %s{two_letter_country_code}\nExample: %sno\n", InfoPath, InfoPath)
		return
	}

	info, st, err := e.Countries.GetCountryInfo(r, code)
	if err != nil {
		http.Error(w, err.Error(), st)
		return
	}

	WriteJSON(w, http.StatusOK, InfoResponse{
		Name:       info.Name,
		Continents: info.Continents,
		Population: info.Population,
		Area:       info.Area,
		Languages:  info.Languages,
		Borders:    info.Borders,
		Flag:       info.FlagPNG,
		Capital:    info.Capital,
	})
}

func (e *HandlerEnv) Exchange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	code := ExtractCode(r.URL.Path, ExchangePath)
	if code == "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Use: %s{two_letter_country_code}\nExample: %sno\n", ExchangePath, ExchangePath)
		return
	}

	core, st, err := e.Countries.GetCountryCore(r, code)
	if err != nil {
		http.Error(w, err.Error(), st)
		return
	}

	if len(core.Borders) == 0 {
		WriteJSON(w, http.StatusOK, ExchangeResponse{
			Country:       core.Name,
			BaseCurrency:  core.BaseCurrency,
			ExchangeRates: []ExchangeRateEntry{},
		})
		return
	}

	neighCurByCCA3, st2, err := e.Countries.GetNeighbourCurrenciesByCCA3(r, core.Borders)
	if err != nil {
		http.Error(w, err.Error(), st2)
		return
	}

	rates, st3, err := e.Currency.GetRates(r, core.BaseCurrency)
	if err != nil {
		http.Error(w, err.Error(), st3)
		return
	}

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
			continue
		}
		exchangeRates = append(exchangeRates, ExchangeRateEntry{cur: rate})
	}

	WriteJSON(w, http.StatusOK, ExchangeResponse{
		Country:       core.Name,
		BaseCurrency:  core.BaseCurrency,
		ExchangeRates: exchangeRates,
	})
}

func (e *HandlerEnv) Register(mux *http.ServeMux) {
	mux.HandleFunc(StatusPath, e.Status)
	mux.HandleFunc(InfoPath, e.Info)
	mux.HandleFunc(ExchangePath, e.Exchange)
}
