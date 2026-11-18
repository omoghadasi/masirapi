package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// Zibal v1 result codes returned in the "result" member.
const (
	zibalResultSuccess         = 100
	zibalResultAlreadyVerified = 201
)

// Zibal has a single production host for both live and test merchants. The
// documented test account uses the merchant "zibal", so a sandbox client only
// swaps the merchant and never the endpoint.
const (
	ZibalProductionAPIBase = "https://gateway.zibal.ir/v1"
	ZibalProductionPayBase = "https://gateway.zibal.ir/start"
	ZibalTestMerchant      = "zibal"
)

// ZibalClient talks to the Zibal payment gateway. Zibal notifies the payer's
// browser through callback_url and the merchant MUST call Verify from its own
// server before settling, so a browser-supplied payment status is never
// trusted.
type ZibalClient struct {
	merchant   string
	requestURL string
	verifyURL  string
	payBase    string
	httpClient *http.Client
}

func NewZibalClient(merchantID string, sandbox bool) *ZibalClient {
	if sandbox {
		merchantID = ZibalTestMerchant
	}
	return &ZibalClient{
		merchant:   merchantID,
		requestURL: ZibalProductionAPIBase + "/request",
		verifyURL:  ZibalProductionAPIBase + "/verify",
		payBase:    ZibalProductionPayBase,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

// ZibalCheckout describes an accepted payment request.
type ZibalCheckout struct {
	TrackID int64
	PayURL  string
}

// ZibalVerification describes a verified payment.
type ZibalVerification struct {
	RefNumber       int64
	CardNumber      string
	Amount          int64
	PaidAt          string
	AlreadyVerified bool
}

// ZibalError reports a gateway rejection with enough context to log, never
// including the merchant credential.
type ZibalError struct {
	Endpoint   string
	HTTPStatus int
	Result     int
	Message    string
}

func (e *ZibalError) Error() string {
	return fmt.Sprintf("zibal %s rejected the request: http_status=%d result=%d message=%q", e.Endpoint, e.HTTPStatus, e.Result, e.Message)
}

type zibalResponse struct {
	TrackID    int64  `json:"trackId"`
	Result     int    `json:"result"`
	Message    string `json:"message"`
	PaidAt     string `json:"paidAt"`
	CardNumber string `json:"cardNumber"`
	Amount     int64  `json:"amount"`
	RefNumber  int64  `json:"refNumber"`
}

// Request reserves a payment for amountRial and returns the gateway URL the
// payer's browser must be sent to.
func (c *ZibalClient) Request(ctx context.Context, amountRial int64, description string, callbackURL string) (*ZibalCheckout, error) {
	payload, err := c.post(ctx, "request", c.requestURL, map[string]any{
		"merchant":    c.merchant,
		"amount":      amountRial,
		"description": description,
		"callbackUrl": callbackURL,
	})
	if err != nil {
		return nil, err
	}
	if payload.TrackID == 0 {
		return nil, &ZibalError{Endpoint: "request", Result: payload.Result, Message: "the gateway returned no trackId"}
	}
	return &ZibalCheckout{
		TrackID: payload.TrackID,
		PayURL:  fmt.Sprintf("%s/%d", c.payBase, payload.TrackID),
	}, nil
}

// Verify confirms the payment server-side. A gateway answer of
// "already verified" is reported as a success with AlreadyVerified set,
// because the money has been captured either way.
func (c *ZibalClient) Verify(ctx context.Context, trackID int64) (*ZibalVerification, error) {
	payload, err := c.post(ctx, "verify", c.verifyURL, map[string]any{
		"merchant": c.merchant,
		"trackId":  trackID,
	})
	if err != nil {
		return nil, err
	}
	return &ZibalVerification{
		RefNumber:       payload.RefNumber,
		CardNumber:      payload.CardNumber,
		Amount:          payload.Amount,
		PaidAt:          payload.PaidAt,
		AlreadyVerified: payload.Result == zibalResultAlreadyVerified,
	}, nil
}

func (c *ZibalClient) post(ctx context.Context, endpoint string, url string, fields map[string]any) (*zibalResponse, error) {
	body, err := common.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("zibal failed to encode %s request: %w", endpoint, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("zibal failed to build %s request: %w", endpoint, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zibal %s request failed: %w", endpoint, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("zibal failed to read %s response: %w", endpoint, err)
	}

	var payload zibalResponse
	if err := common.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("zibal %s response is not valid json: http_status=%d", endpoint, resp.StatusCode)
	}
	if payload.Result != zibalResultSuccess && payload.Result != zibalResultAlreadyVerified {
		message := strings.TrimSpace(payload.Message)
		if message == "" {
			message = strings.TrimSpace(string(raw))
		}
		return nil, &ZibalError{Endpoint: endpoint, HTTPStatus: resp.StatusCode, Result: payload.Result, Message: message}
	}
	return &payload, nil
}
