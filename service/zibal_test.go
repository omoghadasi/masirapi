package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// zibalStubCall records what the client actually put on the wire, because a
// wrong amount unit or a missing merchant is invisible in the response.
type zibalStubCall struct {
	Path        string
	Merchant    string `json:"merchant"`
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
	CallbackURL string `json:"callbackUrl"`
	TrackID     int64  `json:"trackId"`
}

func (c *zibalStubCall) decode(body io.Reader) error {
	raw, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	return common.Unmarshal(raw, c)
}

func newZibalStubServer(t *testing.T, calls *[]zibalStubCall, respond func(path string, w http.ResponseWriter)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := zibalStubCall{Path: r.URL.Path}
		require.NoError(t, call.decode(r.Body))
		*calls = append(*calls, call)
		w.Header().Set("Content-Type", "application/json")
		respond(r.URL.Path, w)
	}))
}

func newZibalStubClient(server *httptest.Server) *ZibalClient {
	return &ZibalClient{
		merchant:   "merchant-under-test",
		requestURL: server.URL + "/v1/request",
		verifyURL:  server.URL + "/v1/verify",
		payBase:    server.URL + "/start",
		httpClient: server.Client(),
	}
}

func TestZibalClientRequestAndVerify(t *testing.T) {
	var calls []zibalStubCall
	server := newZibalStubServer(t, &calls, func(path string, w http.ResponseWriter) {
		switch path {
		case "/v1/request":
			_, _ = w.Write([]byte(`{"trackId":15966442233311,"result":100,"message":"success"}`))
		case "/v1/verify":
			_, _ = w.Write([]byte(`{"paidAt":"2026-11-18T10:00:00","amount":125000,"result":100,"status":1,"refNumber":987654321,"cardNumber":"62741****44","message":"success"}`))
		default:
			t.Errorf("unexpected request path %q", path)
		}
	})
	defer server.Close()

	client := newZibalStubClient(server)
	ctx := context.Background()

	checkout, err := client.Request(ctx, 125000, "Top-up user_id=7", "https://shop.example.com/api/user/zibal/callback")
	require.NoError(t, err)
	assert.Equal(t, int64(15966442233311), checkout.TrackID)
	assert.Equal(t, server.URL+"/start/15966442233311", checkout.PayURL)

	verification, err := client.Verify(ctx, checkout.TrackID)
	require.NoError(t, err)
	assert.Equal(t, int64(987654321), verification.RefNumber)
	assert.Equal(t, "62741****44", verification.CardNumber)
	assert.Equal(t, int64(125000), verification.Amount)
	assert.False(t, verification.AlreadyVerified)

	require.Len(t, calls, 2)
	assert.Equal(t, zibalStubCall{
		Path:        "/v1/request",
		Merchant:    "merchant-under-test",
		Amount:      125000,
		Description: "Top-up user_id=7",
		CallbackURL: "https://shop.example.com/api/user/zibal/callback",
	}, calls[0])
	assert.Equal(t, zibalStubCall{
		Path:     "/v1/verify",
		Merchant: "merchant-under-test",
		TrackID:  15966442233311,
	}, calls[1])
}

// The gateway answers 201 when the same trackId is verified twice. The money
// was captured, so the caller must treat it as a success rather than a failure.
func TestZibalClientReportsAlreadyVerified(t *testing.T) {
	var calls []zibalStubCall
	server := newZibalStubServer(t, &calls, func(path string, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"result":201,"refNumber":112233,"amount":1000,"message":"already verified"}`))
	})
	defer server.Close()

	verification, err := newZibalStubClient(server).Verify(context.Background(), 15966442233311)
	require.NoError(t, err)
	assert.True(t, verification.AlreadyVerified)
	assert.Equal(t, int64(112233), verification.RefNumber)
}

// A rejected request carries a result code and message. The failure must reach
// the log without the merchant credential.
func TestZibalClientRejectsGatewayFailure(t *testing.T) {
	var calls []zibalStubCall
	server := newZibalStubServer(t, &calls, func(path string, w http.ResponseWriter) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"result":105,"message":"amount must be at least 1000 rial"}`))
	})
	defer server.Close()

	_, err := newZibalStubClient(server).Request(context.Background(), 10, "too low", "https://shop.example.com/callback")
	require.Error(t, err)

	var gatewayErr *ZibalError
	require.ErrorAs(t, err, &gatewayErr)
	assert.Equal(t, "request", gatewayErr.Endpoint)
	assert.Equal(t, http.StatusBadRequest, gatewayErr.HTTPStatus)
	assert.Equal(t, 105, gatewayErr.Result)
	assert.Equal(t, "amount must be at least 1000 rial", gatewayErr.Message)
	assert.NotContains(t, err.Error(), "merchant-under-test")
}

// The sandbox is the documented test merchant on the production endpoints.
func TestZibalSandboxUsesTestMerchant(t *testing.T) {
	client := NewZibalClient("live-merchant", true)
	assert.Equal(t, ZibalTestMerchant, client.merchant)
	assert.Equal(t, ZibalProductionAPIBase+"/request", client.requestURL)

	assert.Equal(t, "merchant-id-under-test", NewZibalClient("merchant-id-under-test", false).merchant)
}

// A Toman-priced site must reach the gateway in Rial, and a Rial-priced site
// must not be multiplied again.
func TestZibalRialAmountUsesConfiguredUnit(t *testing.T) {
	zibal := setting.GetZibalSetting()
	originalUnit := zibal.AmountUnit
	t.Cleanup(func() { zibal.AmountUnit = originalUnit })

	testCases := []struct {
		name       string
		unit       string
		price      float64
		wantAmount int64
	}{
		{"toman is ten rial", setting.ZibalAmountUnitToman, 1250, 12500},
		{"toman rounds to whole rial", setting.ZibalAmountUnitToman, 12.5, 125},
		{"toman below one rial", setting.ZibalAmountUnitToman, 0.04, 0},
		{"rial is billed as is", setting.ZibalAmountUnitRial, 1250, 1250},
		{"rial rounds half away from zero", setting.ZibalAmountUnitRial, 1250.5, 1251},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			zibal.AmountUnit = tc.unit
			assert.Equal(t, tc.wantAmount, setting.ZibalRialAmount(tc.price))
		})
	}
}
