package setting

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/config"

	"github.com/shopspring/decimal"
)

// Units accepted by zibal_setting.amount_unit. Zibal always bills the amount
// in Rial, so this only declares which unit the site price (option "Price",
// and therefore TopUp.Money) is expressed in.
const (
	ZibalAmountUnitToman = "toman"
	ZibalAmountUnitRial  = "rial"
)

// ZibalSetting is persisted through the generic "zibal_setting.*" option keys,
// so this gateway needs no option-key registration in model/option.go.
type ZibalSetting struct {
	Enabled bool `json:"enabled"`
	// MerchantID is the gateway credential; it must never be logged.
	MerchantID string `json:"merchant_id"`
	// Sandbox switches to Zibal's documented test merchant ("zibal"), which
	// runs on the same production endpoints.
	Sandbox    bool   `json:"sandbox"`
	AmountUnit string `json:"amount_unit"`
	// MinTopUp is the smallest top-up quantity the gateway may be asked for,
	// in the unit the recharge form uses, like StripeMinTopUp and WaffoMinTopUp.
	MinTopUp int64 `json:"min_topup"`
}

var zibalSetting = ZibalSetting{
	Enabled:    false,
	Sandbox:    false,
	AmountUnit: ZibalAmountUnitToman,
	MinTopUp:   100,
}

func init() {
	config.GlobalConfig.Register("zibal_setting", &zibalSetting)
}

func GetZibalSetting() *ZibalSetting {
	return &zibalSetting
}

func IsZibalEnabled() bool {
	if !zibalSetting.Enabled {
		return false
	}
	// The test merchant is built in, so sandbox mode needs no merchant ID.
	return zibalSetting.Sandbox || strings.TrimSpace(zibalSetting.MerchantID) != ""
}

// ZibalRialAmount converts a price given in the configured site unit to the
// Rial amount the gateway bills. Rounding is half away from zero.
func ZibalRialAmount(price float64) int64 {
	amount := decimal.NewFromFloat(price)
	if !strings.EqualFold(strings.TrimSpace(zibalSetting.AmountUnit), ZibalAmountUnitRial) {
		amount = amount.Mul(tomanInRial)
	}
	return amount.Round(0).IntPart()
}
