package api

type StatusResponse struct {
	CountriesAPI  any    `json:"restcountriesapi"` // spec-required key
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

type ExchangeRateEntry map[string]float64

type ExchangeResponse struct {
	Country       string              `json:"country"`
	BaseCurrency  string              `json:"base-currency"`
	ExchangeRates []ExchangeRateEntry `json:"exchange-rates"`
}
