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

// zarinpalStubCall records what the client actually put on the wire, because a
// wrong amount unit or a missing merchant_id is invisible in the response.
type zarinpalStubCall struct {
	Path        string
	MerchantID  string `json:"merchant_id"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
	CallbackURL string `json:"callback_url"`
	Authority   string `json:"authority"`
}

func (c *zarinpalStubCall) decode(body io.Reader) error {
	raw, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	return common.Unmarshal(raw, c)
}

func newZarinpalStubServer(t *testing.T, calls *[]zarinpalStubCall, respond func(path string, w http.ResponseWriter)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := zarinpalStubCall{Path: r.URL.Path}
		require.NoError(t, call.decode(r.Body))
		*calls = append(*calls, call)
		w.Header().Set("Content-Type", "application/json")
		respond(r.URL.Path, w)
	}))
}

func newZarinpalStubClient(server *httptest.Server) *ZarinpalClient {
	return &ZarinpalClient{
		merchantID: "merchant-id-under-test",
		requestURL: server.URL + "/pg/v4/payment/request.json",
		verifyURL:  server.URL + "/pg/v4/payment/verify.json",
		payBase:    server.URL + "/pg/StartPay",
		httpClient: server.Client(),
	}
}

func TestZarinpalClientRequestAndVerify(t *testing.T) {
	var calls []zarinpalStubCall
	server := newZarinpalStubServer(t, &calls, func(path string, w http.ResponseWriter) {
		switch path {
		case "/pg/v4/payment/request.json":
			_, _ = w.Write([]byte(`{"data":{"code":100,"message":"success","authority":"AUTHORITY-1","fee":0.01,"fee_type":"cappay"},"errors":[]}`))
		case "/pg/v4/payment/verify.json":
			_, _ = w.Write([]byte(`{"data":{"code":100,"message":"success","card_pan":"1234****5678","ref_id":987654321,"fee":0.01,"fee_type":"cappay"},"errors":[]}`))
		default:
			t.Errorf("unexpected request path %q", path)
		}
	})
	defer server.Close()

	client := newZarinpalStubClient(server)
	ctx := context.Background()

	checkout, err := client.Request(ctx, 125000, "Top-up user_id=7", "https://shop.example.com/api/user/zarinpal/callback")
	require.NoError(t, err)
	assert.Equal(t, "AUTHORITY-1", checkout.Authority)
	assert.Equal(t, server.URL+"/pg/StartPay/AUTHORITY-1", checkout.PayURL)
	assert.Equal(t, 0.01, checkout.Fee)
	assert.Equal(t, "cappay", checkout.FeeType)

	verification, err := client.Verify(ctx, "AUTHORITY-1", 125000)
	require.NoError(t, err)
	assert.Equal(t, int64(987654321), verification.RefID)
	assert.Equal(t, "1234****5678", verification.CardPan)
	assert.False(t, verification.AlreadyVerified)

	require.Len(t, calls, 2)
	assert.Equal(t, zarinpalStubCall{
		Path:        "/pg/v4/payment/request.json",
		MerchantID:  "merchant-id-under-test",
		Amount:      125000,
		Currency:    "IRR",
		Description: "Top-up user_id=7",
		CallbackURL: "https://shop.example.com/api/user/zarinpal/callback",
	}, calls[0])
	assert.Equal(t, zarinpalStubCall{
		Path:       "/pg/v4/payment/verify.json",
		MerchantID: "merchant-id-under-test",
		Amount:     125000,
		Authority:  "AUTHORITY-1",
	}, calls[1])
}

// The gateway answers 101 when the same authority is verified twice. The money
// was captured, so the caller must treat it as a success rather than a failure.
func TestZarinpalClientReportsAlreadyVerified(t *testing.T) {
	var calls []zarinpalStubCall
	server := newZarinpalStubServer(t, &calls, func(path string, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"data":{"code":101,"message":"the payment is already verified","ref_id":112233},"errors":[]}`))
	})
	defer server.Close()

	verification, err := newZarinpalStubClient(server).Verify(context.Background(), "AUTHORITY-1", 1000)
	require.NoError(t, err)
	assert.True(t, verification.AlreadyVerified)
	assert.Equal(t, int64(112233), verification.RefID)
}

// Gateway failures use HTTP 4xx and an "errors" object, while success uses an
// "errors" array. The failure message must reach the log without the credential.
func TestZarinpalClientRejectsGatewayFailure(t *testing.T) {
	var calls []zarinpalStubCall
	server := newZarinpalStubServer(t, &calls, func(path string, w http.ResponseWriter) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":{"message":"invalid authority","code":2009}}`))
	})
	defer server.Close()

	_, err := newZarinpalStubClient(server).Verify(context.Background(), "BAD-AUTHORITY", 1000)
	require.Error(t, err)

	var gatewayErr *ZarinpalError
	require.ErrorAs(t, err, &gatewayErr)
	assert.Equal(t, "verify", gatewayErr.Endpoint)
	assert.Equal(t, http.StatusUnprocessableEntity, gatewayErr.HTTPStatus)
	assert.Equal(t, "invalid authority", gatewayErr.Message)
	assert.NotContains(t, err.Error(), "merchant-id-under-test")
}

// A Toman-priced site must reach the gateway in Rial, and a Rial-priced site
// must not be multiplied again.
func TestZarinpalRialAmountUsesConfiguredUnit(t *testing.T) {
	zarinpal := setting.GetZarinpalSetting()
	originalUnit := zarinpal.AmountUnit
	t.Cleanup(func() { zarinpal.AmountUnit = originalUnit })

	testCases := []struct {
		name       string
		unit       string
		price      float64
		wantAmount int64
	}{
		{"toman is ten rial", setting.ZarinpalAmountUnitToman, 1250, 12500},
		{"toman rounds to whole rial", setting.ZarinpalAmountUnitToman, 12.5, 125},
		{"toman below one rial", setting.ZarinpalAmountUnitToman, 0.04, 0},
		{"rial is billed as is", setting.ZarinpalAmountUnitRial, 1250, 1250},
		{"rial rounds half away from zero", setting.ZarinpalAmountUnitRial, 1250.5, 1251},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			zarinpal.AmountUnit = tc.unit
			assert.Equal(t, tc.wantAmount, setting.ZarinpalRialAmount(tc.price))
		})
	}
}
