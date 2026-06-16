// Package market provides stock and crypto market data via Yahoo Finance's
// unofficial API. It requires no API key and supports IDX stocks (.JK suffix)
// as well as global equities and cryptocurrencies.
package market

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Quote represents a single security's snapshot.
type Quote struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
	High          float64 `json:"high"`
	Low           float64 `json:"low"`
	Open          float64 `json:"open"`
	Volume        int64   `json:"volume"`
	Currency      string  `json:"currency"`
	Exchange      string  `json:"exchange"`
}

// OHLC is one candlestick bar.
type OHLC struct {
	Time   int64   `json:"time"` // unix seconds
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
}

// --- Public API ------------------------------------------------------------

func FetchQuote(symbol string) (*Quote, error) {
	quotes, err := FetchMultiQuotes([]string{symbol})
	if err != nil {
		return nil, err
	}
	if len(quotes) == 0 {
		return nil, fmt.Errorf("no data for %s", symbol)
	}
	return &quotes[0], nil
}

func FetchMultiQuotes(symbols []string) ([]Quote, error) {
	var quotes []Quote
	for _, sym := range symbols {
		q, err := fetchOne(sym)
		if err != nil {
			continue // skip symbols that fail
		}
		quotes = append(quotes, *q)
	}
	return quotes, nil
}

func FetchChart(symbol, rng string) ([]OHLC, error) {
	if rng == "" {
		rng = "1mo"
	}
	return fetchChart(symbol, rng)
}

// --- Internal ---------------------------------------------------------------

var httpClient = &http.Client{Timeout: 10 * time.Second}

type chartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol              string  `json:"symbol"`
				RegularMarketPrice  float64 `json:"regularMarketPrice"`
				RegularMarketChange float64 `json:"regularMarketChange"`
				RegularMarketDayHigh float64 `json:"regularMarketDayHigh"`
				RegularMarketDayLow  float64 `json:"regularMarketDayLow"`
				RegularMarketVolume  int64   `json:"regularMarketVolume"`
				ChartPreviousClose   float64 `json:"chartPreviousClose"`
				Currency             string  `json:"currency"`
				ExchangeName         string  `json:"exchangeName"`
				ShortName            string  `json:"shortName"`
				LongName             string  `json:"longName"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []float64 `json:"open"`
					High   []float64 `json:"high"`
					Low    []float64 `json:"low"`
					Close  []float64 `json:"close"`
					Volume []int64   `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}

func fetchOne(symbol string) (*Quote, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?range=2d&interval=1d", symbol)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("yahoo finance %d for %s", resp.StatusCode, symbol)
	}

	var out chartResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Chart.Result) == 0 {
		return nil, fmt.Errorf("no result for %s", symbol)
	}

	r := out.Chart.Result[0]
	m := r.Meta
	name := m.ShortName
	if name == "" {
		name = m.LongName
	}
	if name == "" {
		name = symbol
	}

	// Get latest OHLC from quote data
	var o float64
	quote := r.Indicators.Quote[0]
	last := len(quote.Close) - 1
	if last < 0 {
		last = 0
	}
	for last >= 0 && quote.Close[last] == 0 {
		last--
	}
	if last >= 0 {
		o = quote.Open[last]
	}

	ch := m.RegularMarketPrice - m.ChartPreviousClose
	chPct := 0.0
	if m.ChartPreviousClose > 0 {
		chPct = ch / m.ChartPreviousClose * 100
	}

	curr := m.Currency
	if curr == "" && strings.HasSuffix(symbol, ".JK") {
		curr = "IDR"
	}

	return &Quote{
		Symbol:        m.Symbol,
		Name:          name,
		Price:         m.RegularMarketPrice,
		Change:        ch,
		ChangePercent: chPct,
		High:          m.RegularMarketDayHigh,
		Low:           m.RegularMarketDayLow,
		Open:          o,
		Volume:        m.RegularMarketVolume,
		Currency:      curr,
		Exchange:      m.ExchangeName,
	}, nil
}

func fetchChart(symbol, rng string) ([]OHLC, error) {
	interval := "1d"
	switch rng {
	case "1d":
		interval = "5m"
	case "5d":
		interval = "15m"
	}

	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?range=%s&interval=%s", symbol, rng, interval)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("yahoo finance %d for %s", resp.StatusCode, symbol)
	}

	var out chartResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Chart.Result) == 0 {
		return nil, fmt.Errorf("no chart data for %s", symbol)
	}

	r := out.Chart.Result[0]
	q := r.Indicators.Quote[0]

	var result []OHLC
	for i := range r.Timestamp {
		o := q.Open[i]
		h := q.High[i]
		l := q.Low[i]
		c := q.Close[i]
		v := q.Volume[i]
		if o == 0 && c == 0 {
			continue // skip null entries
		}
		result = append(result, OHLC{
			Time:   r.Timestamp[i],
			Open:   o,
			High:   h,
			Low:    l,
			Close:  c,
			Volume: v,
		})
	}
	return result, nil
}
