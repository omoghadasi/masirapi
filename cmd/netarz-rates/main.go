// Command netarz-rates keeps the site's three USD/Toman rate options in step with
// the netarz.ir FX API. It ships in the same image as the API and runs as its own
// container (docker-compose service "netarz-rates"), so the API binary is untouched.
//
// Reads NETARZ_API_KEY and SQL_DSN from the environment, syncs once, then every 30 min.
// ponytail: postgres only - SQL_DSN in this deployment is postgres; add a mysql branch if that changes.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// The "Toman per USD" knobs in System Settings -> Billing & Payment.
var rateKeys = []string{
	"general_setting.custom_currency_exchange_rate", // Currency & Display: Units per USD
	"Price",           // Payment Gateway: Price (local currency / USD)
	"USDExchangeRate", // Currency & Display: USD Exchange Rate
}

const (
	defaultAPI = "https://netarz.ir/api/fx/v1/rates?codes=USD"
	interval   = 30 * time.Minute
	// Sanity band: an outside API writes money values here, so refuse anything wild.
	minRate = 50_000
	maxRate = 5_000_000
)

func main() {
	apiURL := os.Getenv("NETARZ_FX_URL")
	if apiURL == "" {
		apiURL = defaultAPI
	}
	apiKey := os.Getenv("NETARZ_API_KEY")
	if apiKey == "" {
		log.Fatal("NETARZ_API_KEY is not set")
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
		if err := sync(db, apiURL, apiKey); err != nil {
			log.Printf("netarz: %v", err)
		}
		time.Sleep(interval)
	}
}

func sync(db *sql.DB, apiURL, apiKey string) error {
	rate, err := fetchRate(apiURL, apiKey)
	if err != nil {
		return err
	}
	value := strconv.FormatFloat(rate, 'f', -1, 64)
	for _, key := range rateKeys {
		// Upsert: the option row may not exist yet on a fresh database.
		if _, err := db.Exec(`INSERT INTO options (key, value) VALUES ($1, $2)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, key, value); err != nil {
			return fmt.Errorf("write %s: %w", key, err)
		}
	}
	log.Printf("USD/Toman = %s -> %v", value, rateKeys)
	return nil
}

func fetchRate(apiURL, apiKey string) (float64, error) {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP %d %s", resp.StatusCode, body)
	}

	var payload struct {
		Data []struct {
			Code string  `json:"code"`
			Mid  float64 `json:"mid"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, fmt.Errorf("bad payload: %w", err)
	}
	for _, r := range payload.Data {
		if r.Code != "USD" {
			continue
		}
		if r.Mid <= minRate || r.Mid >= maxRate {
			return 0, fmt.Errorf("implausible USD rate %v, refusing to write", r.Mid)
		}
		return r.Mid, nil
	}
	return 0, fmt.Errorf("no USD rate in %s", body)
}
