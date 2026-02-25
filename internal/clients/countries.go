package clients

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type CountriesClient struct {
	BaseURL string
	HTTP    *http.Client
}

type currencyInfo struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
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

	CCA3 string `json:"cca3"`

	Currencies map[string]currencyInfo `json:"currencies"`
}

type countryAlphaResponse []countryAlphaItem

func decodeCountryAlpha(resp *http.Response) (countryAlphaItem, int, error) {
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return countryAlphaItem{}, http.StatusBadGateway, err
	}

	var arr countryAlphaResponse
	if err := json.Unmarshal(raw, &arr); err == nil {
		if len(arr) == 0 {
			return countryAlphaItem{}, http.StatusNotFound, fmt.Errorf("no country data")
		}
		return arr[0], http.StatusOK, nil
	}

	var obj countryAlphaItem
	if err := json.Unmarshal(raw, &obj); err == nil {
		if obj.Name.Common == "" {
			return countryAlphaItem{}, http.StatusBadGateway, fmt.Errorf("unexpected country payload")
		}
		return obj, http.StatusOK, nil
	}

	return countryAlphaItem{}, http.StatusBadGateway, fmt.Errorf("could not decode country payload")
}

func pickDeterministicCurrencyCode(m map[string]currencyInfo) string {
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

type CountryInfo struct {
	Name       string
	Continents []string
	Population int
	Area       float64
	Languages  map[string]string
	Borders    []string
	FlagPNG    string
	Capital    string
}

func (c *CountriesClient) GetCountryInfo(reqCtx *http.Request, twoLetter string) (CountryInfo, int, error) {
	twoLetter = strings.ToLower(strings.TrimSpace(twoLetter))
	if len(twoLetter) != 2 {
		return CountryInfo{}, http.StatusBadRequest, fmt.Errorf("country code must be 2 letters")
	}

	u, _ := url.Parse(strings.TrimRight(c.BaseURL, "/") + "/v3.1/alpha/" + twoLetter)
	q := u.Query()
	q.Set("fields", "name,continents,population,area,languages,borders,flags,capital")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(reqCtx.Context(), http.MethodGet, u.String(), nil)
	if err != nil {
		return CountryInfo{}, http.StatusInternalServerError, err
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return CountryInfo{}, http.StatusBadGateway, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return CountryInfo{}, resp.StatusCode, fmt.Errorf("countries api returned %d", resp.StatusCode)
	}

	item, st, err := decodeCountryAlpha(resp)
	if err != nil {
		return CountryInfo{}, st, err
	}

	capital := ""
	if len(item.Capital) > 0 {
		capital = item.Capital[0]
	}

	return CountryInfo{
		Name:       item.Name.Common,
		Continents: item.Continents,
		Population: item.Population,
		Area:       item.Area,
		Languages:  item.Languages,
		Borders:    item.Borders,
		FlagPNG:    item.Flags.PNG,
		Capital:    capital,
	}, http.StatusOK, nil
}

type CountryCore struct {
	Name         string
	Borders      []string
	BaseCurrency string
}

func (c *CountriesClient) GetCountryCore(reqCtx *http.Request, twoLetter string) (CountryCore, int, error) {
	twoLetter = strings.ToLower(strings.TrimSpace(twoLetter))
	if len(twoLetter) != 2 {
		return CountryCore{}, http.StatusBadRequest, fmt.Errorf("country code must be 2 letters")
	}

	u, _ := url.Parse(strings.TrimRight(c.BaseURL, "/") + "/v3.1/alpha/" + twoLetter)
	q := u.Query()
	q.Set("fields", "name,borders,currencies")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(reqCtx.Context(), http.MethodGet, u.String(), nil)
	if err != nil {
		return CountryCore{}, http.StatusInternalServerError, err
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return CountryCore{}, http.StatusBadGateway, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return CountryCore{}, resp.StatusCode, fmt.Errorf("countries api returned %d", resp.StatusCode)
	}

	item, st, err := decodeCountryAlpha(resp)
	if err != nil {
		return CountryCore{}, st, err
	}

	baseCur := pickDeterministicCurrencyCode(item.Currencies)
	if baseCur == "" {
		return CountryCore{}, http.StatusBadGateway, fmt.Errorf("missing currency for country")
	}

	return CountryCore{
		Name:         item.Name.Common,
		Borders:      item.Borders,
		BaseCurrency: baseCur,
	}, http.StatusOK, nil
}

func (c *CountriesClient) GetNeighbourCurrenciesByCCA3(reqCtx *http.Request, cca3Codes []string) (map[string]string, int, error) {
	clean := make([]string, 0, len(cca3Codes))
	seen := map[string]bool{}
	for _, x := range cca3Codes {
		x = strings.ToUpper(strings.TrimSpace(x))
		if len(x) != 3 || seen[x] {
			continue
		}
		seen[x] = true
		clean = append(clean, x)
	}
	if len(clean) == 0 {
		return map[string]string{}, http.StatusOK, nil
	}

	u, _ := url.Parse(strings.TrimRight(c.BaseURL, "/") + "/v3.1/alpha")
	q := u.Query()
	q.Set("codes", strings.Join(clean, ","))
	q.Set("fields", "cca3,currencies")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(reqCtx.Context(), http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("countries api returned %d", resp.StatusCode)
	}

	var payload []countryAlphaItem
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, http.StatusBadGateway, err
	}

	out := make(map[string]string, len(payload))
	for _, n := range payload {
		out[n.CCA3] = pickDeterministicCurrencyCode(n.Currencies)
	}
	return out, http.StatusOK, nil
}
