package setting

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/config"

	"github.com/shopspring/decimal"
)

// Units accepted by zarinpal_setting.amount_unit. ZarinPal always bills the
// amount in Rial, so this only declares which unit the site price (option
// "Price", and therefore TopUp.Money) is expressed in.
const (
	ZarinpalAmountUnitToman = "toman"
	ZarinpalAmountUnitRial  = "rial"
)

var tomanInRial = decimal.NewFromInt(10)

// ZarinpalSetting is persisted through the generic "zarinpal_setting.*" option
// keys, so this gateway needs no option-key registration in model/option.go.
type ZarinpalSetting struct {
	Enabled bool `json:"enabled"`
	// MerchantID is the gateway credential; it must never be logged.
	MerchantID string `json:"merchant_id"`
	Sandbox    bool   `json:"sandbox"`
	AmountUnit string `json:"amount_unit"`
	// MinTopUp is the smallest top-up quantity the gateway may be asked for,
	// in the unit the recharge form uses, like StripeMinTopUp and WaffoMinTopUp.
	MinTopUp int64 `json:"min_topup"`
}

var zarinpalSetting = ZarinpalSetting{
	Enabled:    false,
	Sandbox:    false,
	AmountUnit: ZarinpalAmountUnitToman,
	MinTopUp:   1,
}

func init() {
	config.GlobalConfig.Register("zarinpal_setting", &zarinpalSetting)
}

func GetZarinpalSetting() *ZarinpalSetting {
	return &zarinpalSetting
}

func IsZarinpalEnabled() bool {
	return zarinpalSetting.Enabled && strings.TrimSpace(zarinpalSetting.MerchantID) != ""
}

// ZarinpalRialAmount converts a price given in the configured site unit to the
// Rial amount the gateway bills. Rounding is half away from zero.
func ZarinpalRialAmount(price float64) int64 {
	amount := decimal.NewFromFloat(price)
	if !strings.EqualFold(strings.TrimSpace(zarinpalSetting.AmountUnit), ZarinpalAmountUnitRial) {
		amount = amount.Mul(tomanInRial)
	}
	return amount.Round(0).IntPart()
}
