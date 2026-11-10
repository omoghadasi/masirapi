// Command wallex-rates keeps the site's three USD/Toman rate options in step with
// the Wallex USDT/Toman market (USDTTMN). It ships in the same image as the API and
// runs as its own container (docker-compose service "wallex-rates"), so the API
// binary is untouched. Reads SQL_DSN from the environment, syncs once, then every
// 30 min. Wallex's public markets endpoint needs no credentials.
//
// ponytail: postgres only - SQL_DSN in this deployment is postgres; add a mysql branch if that changes.
package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// The "Toman per USD" knobs in System Settings -> Billing & Payment.
var rateKeys = []string{
	"general_setting.custom_currency_exchange_rate", // Currency & Display: Units per USD
	"Price",           // Payment Gateway: Price (local currency / USD)
	"USDExchangeRate", // Currency & Display: USD Exchange Rate
}

const (
	defaultMarketsURL = "https://api.wallex.ir/v1/markets"
	symbol            = "USDTTMN" // Tether - Toman
	interval          = 30 * time.Minute
	// Sanity band: an outside API writes money values here, so refuse anything wild.
	minRate = 50_000
	maxRate = 5_000_000
)

func main() {
	marketsURL := os.Getenv("WALLEX_MARKETS_URL")
	if marketsURL == "" {
		marketsURL = defaultMarketsURL
	}
	markupPercent := 0.0
	if raw := os.Getenv("WALLEX_RATE_MARKUP_PERCENT"); raw != "" {
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil || value <= -100 {
			log.Fatalf("WALLEX_RATE_MARKUP_PERCENT must be a number greater than -100: %q", raw)
		}
		markupPercent = value
	}
	dsn := os.Getenv("SQL_DSN")
	if dsn == "" {
		log.Fatal("SQL_DSN is not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	for {
		if err := sync(db, marketsURL, markupPercent); err != nil {
			log.Printf("wallex: %v", err)
		}
		time.Sleep(interval)
	}
}

func sync(db *sql.DB, marketsURL string, markupPercent float64) error {
	rate, err := fetchRate(marketsURL)
	if err != nil {
		return err
	}
	rate *= 1 + markupPercent/100
	if rate <= minRate || rate >= maxRate {
		return fmt.Errorf("rate %v after %g%% markup is outside the sanity band, refusing to write", rate, markupPercent)
	}
	value := strconv.FormatFloat(rate, 'f', -1, 64)
	for _, key := range rateKeys {
		// Upsert: the option row may not exist yet on a fresh database.
		if _, err := db.Exec(`INSERT INTO options (key, value) VALUES ($1, $2)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, key, value); err != nil {
			return fmt.Errorf("write %s: %w", key, err)
		}
	}
	log.Printf("USDT/Toman = %s (markup %g%%) -> %v", value, markupPercent, rateKeys)
	return nil
}

func fetchRate(marketsURL string) (float64, error) {
	req, err := http.NewRequest(http.MethodGet, marketsURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP %d %s", resp.StatusCode, body)
	}

	var payload struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Result  struct {
			Symbols map[string]struct {
				Stats struct {
					LastPrice string `json:"lastPrice"`
				} `json:"stats"`
			} `json:"symbols"`
		} `json:"result"`
	}
	if err := common.Unmarshal(body, &payload); err != nil {
		return 0, fmt.Errorf("bad payload: %w", err)
	}
	if !payload.Success {
		return 0, fmt.Errorf("wallex error: %s", payload.Message)
	}
	market, ok := payload.Result.Symbols[symbol]
	if !ok {
		return 0, fmt.Errorf("no %s market in response", symbol)
	}
	rate, err := strconv.ParseFloat(market.Stats.LastPrice, 64)
	if err != nil {
		return 0, fmt.Errorf("bad lastPrice %q: %w", market.Stats.LastPrice, err)
	}
	if rate <= minRate || rate >= maxRate {
		return 0, fmt.Errorf("implausible %s rate %v, refusing to write", symbol, rate)
	}
	return rate, nil
}
