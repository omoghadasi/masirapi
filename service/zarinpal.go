package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// ZarinPal v4 payment result codes returned in the "data.code" member.
const (
	zarinpalCodeVerified        = 100
	zarinpalCodeAlreadyVerified = 101
)

// The gateway answers with an HTTP 200 and data.code on success, and with a
// 4xx and an "errors" member on failure. "errors" is an object for gateway
// failures but an empty array on success, so it is kept as raw JSON and only
// used to build a human-readable failure description.
const (
	ZarinpalProductionAPIBase = "https://api.zarinpal.com/pg/v4/payment"
	ZarinpalSandboxAPIBase    = "https://sandbox.zarinpal.com/pg/v4/payment"
	ZarinpalProductionPayBase = "https://www.zarinpal.com/pg/StartPay"
	ZarinpalSandboxPayBase    = "https://sandbox.zarinpal.com/pg/StartPay"

	// zarinpalCurrencyRial is always sent on the request so the amount unit
	// never depends on the merchant panel default.
	zarinpalCurrencyRial = "IRR"
)

// ZarinpalClient talks to the ZarinPal payment gateway. The gateway has no
// server-to-server notification: the payer's browser is redirected back to
// callback_url and the merchant MUST call Verify from its own server before
// settling, so a browser-supplied payment status is never trusted.
type ZarinpalClient struct {
	merchantID string
	requestURL string
	verifyURL  string
	payBase    string
	httpClient *http.Client
}

func NewZarinpalClient(merchantID string, sandbox bool) *ZarinpalClient {
	apiBase := ZarinpalProductionAPIBase
	payBase := ZarinpalProductionPayBase
	if sandbox {
		apiBase = ZarinpalSandboxAPIBase
		payBase = ZarinpalSandboxPayBase
	}
	return &ZarinpalClient{
		merchantID: merchantID,
		requestURL: apiBase + "/request.json",
		verifyURL:  apiBase + "/verify.json",
		payBase:    payBase,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

// ZarinpalCheckout describes an accepted payment request.
type ZarinpalCheckout struct {
	Authority string
	PayURL    string
	Fee       float64
	FeeType   string
}

// ZarinpalVerification describes a verified payment.
type ZarinpalVerification struct {
	RefID           int64
	CardPan         string
	Fee             float64
	FeeType         string
	AlreadyVerified bool
}

// ZarinpalError reports a gateway rejection with enough context to log, never
// including the merchant credential.
type ZarinpalError struct {
	Endpoint   string
	HTTPStatus int
	Code       int
	Message    string
}

func (e *ZarinpalError) Error() string {
	return fmt.Sprintf("zarinpal %s rejected the request: http_status=%d code=%d message=%q", e.Endpoint, e.HTTPStatus, e.Code, e.Message)
}

type zarinpalResponse struct {
	Data struct {
		Code      int     `json:"code"`
		Message   string  `json:"message"`
		Authority string  `json:"authority"`
		RefID     int64   `json:"ref_id"`
		CardPan   string  `json:"card_pan"`
		Fee       float64 `json:"fee"`
		FeeType   string  `json:"fee_type"`
	} `json:"data"`
	Errors json.RawMessage `json:"errors"`
}

func (r *zarinpalResponse) failureMessage() string {
	if len(r.Errors) == 0 || common.GetJsonType(r.Errors) != "object" {
		return ""
	}
	var failure struct {
		Message string `json:"message"`
	}
	if common.UnmarshalJsonStr(string(r.Errors), &failure) != nil {
		return ""
	}
	return failure.Message
}

// Request reserves a payment for amountRial and returns the gateway URL the
// payer's browser must be sent to.
func (c *ZarinpalClient) Request(ctx context.Context, amountRial int64, description string, callbackURL string) (*ZarinpalCheckout, error) {
	payload, err := c.post(ctx, "request", map[string]any{
		"merchant_id":  c.merchantID,
		"amount":       amountRial,
		"currency":     zarinpalCurrencyRial,
		"description":  description,
		"callback_url": callbackURL,
	})
	if err != nil {
		return nil, err
	}
	if payload.Data.Authority == "" {
		return nil, &ZarinpalError{Endpoint: "request", Code: payload.Data.Code, Message: "the gateway returned no authority"}
	}
	return &ZarinpalCheckout{
		Authority: payload.Data.Authority,
		PayURL:    c.payBase + "/" + payload.Data.Authority,
		Fee:       payload.Data.Fee,
		FeeType:   payload.Data.FeeType,
	}, nil
}

// Verify confirms the payment server-side. A gateway answer of
// "already verified" is reported as a success with AlreadyVerified set,
// because the money has been captured either way.
func (c *ZarinpalClient) Verify(ctx context.Context, authority string, amountRial int64) (*ZarinpalVerification, error) {
	payload, err := c.post(ctx, "verify", map[string]any{
		"merchant_id": c.merchantID,
		"authority":   authority,
		"amount":      amountRial,
	})
	if err != nil {
		return nil, err
	}
	return &ZarinpalVerification{
		RefID:           payload.Data.RefID,
		CardPan:         payload.Data.CardPan,
		Fee:             payload.Data.Fee,
		FeeType:         payload.Data.FeeType,
		AlreadyVerified: payload.Data.Code == zarinpalCodeAlreadyVerified,
	}, nil
}

func (c *ZarinpalClient) post(ctx context.Context, endpoint string, fields map[string]any) (*zarinpalResponse, error) {
	url := c.requestURL
	if endpoint == "verify" {
		url = c.verifyURL
	}
	body, err := common.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("zarinpal failed to encode %s request: %w", endpoint, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("zarinpal failed to build %s request: %w", endpoint, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zarinpal %s request failed: %w", endpoint, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("zarinpal failed to read %s response: %w", endpoint, err)
	}

	var payload zarinpalResponse
	if err := common.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("zarinpal %s response is not valid json: http_status=%d", endpoint, resp.StatusCode)
	}
	if payload.Data.Code != zarinpalCodeVerified && payload.Data.Code != zarinpalCodeAlreadyVerified {
		message := payload.Data.Message
		if message == "" {
			message = payload.failureMessage()
		}
		if message == "" {
			message = strings.TrimSpace(string(raw))
		}
		return nil, &ZarinpalError{Endpoint: endpoint, HTTPStatus: resp.StatusCode, Code: payload.Data.Code, Message: message}
	}
	return &payload, nil
}
