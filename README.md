# CountryInfo REST API

This project is a REST API written in Go that provides information about countries and currency exchange rates for neighbouring countries.

The service combines data from two external REST APIs:
- a REST Countries API
- a Currency Exchange API

The service does not store any data itself. All information is fetched dynamically when a request is made.

This is my first REST API project and is made using only the Go standard library.

---

## API Version

Current version: **v1**

Base path:

```
/countryinfo/v1/
```

---

## Endpoints

### GET `/countryinfo/v1/status/`

Returns the status of the external APIs and the uptime in seconds of this service.

Example response:
```json
{
  "restcountriesapi": 200,
  "currenciesapi": 200,
  "version": "v1",
  "uptime": 1234
}
```

---

### GET `/countryinfo/v1/info/{country_code}`

Returns general information about a country.

- `{country_code}` must be a 2-letter ISO country code
- Case-insensitive

Example:
```
/countryinfo/v1/info/no
```

Example response:
```json
{
  "name": "Norway",
  "continents": ["Europe"],
  "population": 5379475,
  "area": 323802,
  "languages": {
    "nno": "Norwegian Nynorsk",
    "nob": "Norwegian Bokmål",
    "smi": "Sami"
  },
  "borders": ["FIN", "SWE", "RUS"],
  "flag": "https://flagcdn.com/w320/no.png",
  "capital": "Oslo"
}
```

---

### GET `/countryinfo/v1/exchange/{country_code}`

Returns exchange rates from the country’s base currency to the currencies of neighbouring countries.

Example:
```
/countryinfo/v1/exchange/no
```

Example response:
```json
{
  "country": "Norway",
  "base-currency": "NOK",
  "exchange-rates": [
    { "EUR": 0.08878 },
    { "SEK": 0.946335 },
    { "RUB": 8.011422 }
  ]
}
```

If a country has no neighbours, the exchange rate list will be empty.

---

## Multiple Capitals and Currencies

Some countries have multiple capitals or currencies.

This service follows the original assignment specification and returns:
- one capital (the first provided by the Countries API)
- one base currency

If multiple currencies exist, the alphabetically first currency code is selected deterministically.

---

## Error Handling

- Invalid input → `400 Bad Request`
- Unknown country → `404 Not Found`
- External service failure → `502 Bad Gateway`

Errors are returned as JSON, for example:
```json
{
  "error": "country not found"
}
```

---

## Running Locally

### Requirements
- Go 1.20 or newer

Run from the project root:
```
go run ./cmd/countryinfo
```

The service runs on port **8080**.

---

## External APIs Used

- REST Countries API (self-hosted by the course)
- Currency Exchange API (self-hosted by the course)

All data is retrieved dynamically.

---

## Project Structure

```
cmd/countryinfo/        Application entry point
internal/api/           HTTP handlers and response models
internal/clients/       Clients for external APIs
```

---

## Deployment

The service is intended to be deployed on **Render** using a separate GitHub repository.

Build command:
```
go build -o app ./cmd/countryinfo
```

Start command:
```
./app
```

---
This README was written in cooperation with AI for structure and clarity purposes.
---

Individual assignment in **PROG2005** at **NTNU**
