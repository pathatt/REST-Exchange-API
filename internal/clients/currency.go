package clients

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type CurrencyClient struct {
	BaseURL string
	HTTP    *http.Client
}

type currencyRatesResponse struct {
	Result   string             `json:"result"`
	BaseCode string             `json:"base_code"`
	Rates    map[string]float64 `json:"rates"`
}

func (c *CurrencyClient) GetRates(reqCtx *http.Request, baseCurrency string) (map[string]float64, int, error) {
	baseCurrency = strings.ToUpper(strings.TrimSpace(baseCurrency))
	if len(baseCurrency) != 3 {
		return nil, http.StatusBadRequest, fmt.Errorf("base currency must be 3 letters")
	}

	req, err := http.NewRequestWithContext(reqCtx.Context(), http.MethodGet, strings.TrimRight(c.BaseURL, "/")+"/"+baseCurrency, nil)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	resp, err := c.HTTP.Do(req)
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
	if strings.ToLower(payload.Result) != "success" {
		return nil, http.StatusBadGateway, fmt.Errorf("unexpected currencies payload")
	}

	return payload.Rates, http.StatusOK, nil
}
