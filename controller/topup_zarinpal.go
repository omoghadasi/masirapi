package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// zarinpalCallbackPath must match the public route registered in the router.
const zarinpalCallbackPath = "/api/user/zarinpal/callback"

func isZarinpalTopUpEnabled() bool {
	return isPaymentComplianceConfirmed() && setting.IsZarinpalEnabled()
}

func isZarinpalWebhookEnabled() bool {
	return isZarinpalTopUpEnabled()
}

func GetZarinpalClient() *service.ZarinpalClient {
	if !setting.IsZarinpalEnabled() {
		return nil
	}
	zarinpal := setting.GetZarinpalSetting()
	return service.NewZarinpalClient(strings.TrimSpace(zarinpal.MerchantID), zarinpal.Sandbox)
}

type ZarinpalPayRequest struct {
	// Amount is the top-up quantity in the unit the recharge form uses.
	Amount int64 `json:"amount"`
	// PaymentMethod must be the ZarinPal method.
	PaymentMethod string `json:"payment_method"`
}

// RequestZarinpalPay reserves a gateway payment and returns its checkout URL.
// ZarinPal sends no server-to-server notification, so the order is settled by
// ZarinpalCallback once the gateway confirms the payment server-side.
func RequestZarinpalPay(c *gin.Context) {
	var req ZarinpalPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		paymentError(c, common.NewMessage("Invalid parameters"))
		return
	}
	if req.PaymentMethod != model.PaymentMethodZarinpal {
		paymentError(c, common.NewMessage("Unsupported payment channel"))
		return
	}
	minAmount := max(getMinTopup(), setting.GetZarinpalSetting().MinTopUp)
	if req.Amount < minAmount {
		paymentError(c, common.NewMessage("Top-up amount cannot be less than {{min}}", map[string]any{"min": minAmount}))
		return
	}

	id := c.GetInt("id")
	if rejectInvalidTopUpQuota(c, id, req.Amount) {
		return
	}
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		paymentError(c, common.NewMessage("Failed to get the user group"))
		return
	}
	client := GetZarinpalClient()
	if client == nil {
		paymentError(c, common.NewMessage("Payment information has not been configured by administrator"))
		return
	}

	payMoney := getPayMoney(req.Amount, group)
	amountRial := setting.ZarinpalRialAmount(payMoney)
	if amountRial <= 0 {
		paymentError(c, common.NewMessage("Payment amount is too low"))
		return
	}

	checkout, err := client.Request(c.Request.Context(), amountRial, fmt.Sprintf("Top-up user_id=%d", id), service.GetCallbackAddress()+zarinpalCallbackPath)
	if err != nil {
		logger.LogError(c.Request.Context(), common.LogText("ZarinPal failed to start payment user_id=%d amount=%d error=%q", id, req.Amount, err.Error()))
		paymentError(c, common.NewMessage("Failed to start payment"))
		return
	}

	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount := decimal.NewFromInt(amount)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		amount = dAmount.Div(dQuotaPerUnit).IntPart()
	}
	topUp := &model.TopUp{
		UserId:          id,
		Amount:          amount,
		Money:           payMoney,
		TradeNo:         checkout.Authority,
		PaymentMethod:   model.PaymentMethodZarinpal,
		PaymentProvider: model.PaymentProviderZarinpal,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), common.LogText("ZarinPal failed to create top-up order user_id=%d trade_no=%s amount=%d error=%q", id, checkout.Authority, amount, err.Error()))
		paymentError(c, common.NewMessage("Failed to create order"))
		return
	}
	logger.LogInfo(c.Request.Context(), common.LogText("ZarinPal top-up order created user_id=%d trade_no=%s amount=%d money=%.2f fee=%.2f fee_type=%s", id, checkout.Authority, amount, payMoney, checkout.Fee, checkout.FeeType))
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"pay_link": checkout.PayURL,
		},
	})
}

// ZarinpalCallback handles the browser redirect the gateway performs after the
// payer leaves its payment page. The query status only decides whether the
// gateway is asked to confirm; quota is credited exclusively on a successful
// server-side verification, so a forged Status=OK grants nothing.
func ZarinpalCallback(c *gin.Context) {
	if !isZarinpalWebhookEnabled() {
		logger.LogWarn(c.Request.Context(), common.LogText("ZarinPal callback rejected reason=gateway_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=fail"))
		return
	}

	authority := c.Query("Authority")
	if authority == "" {
		logger.LogWarn(c.Request.Context(), common.LogText("ZarinPal callback authority is missing path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=fail"))
		return
	}

	if !strings.EqualFold(c.Query("Status"), "OK") {
		logger.LogInfo(c.Request.Context(), common.LogText("ZarinPal callback payment not completed trade_no=%s client_ip=%s", authority, c.ClientIP()))
		if err := model.UpdatePendingTopUpStatus(authority, model.PaymentProviderZarinpal, common.TopUpStatusFailed); err != nil && !errors.Is(err, model.ErrTopUpNotFound) {
			logger.LogWarn(c.Request.Context(), common.LogText("ZarinPal callback failed to mark order unpaid trade_no=%s client_ip=%s error=%q", authority, c.ClientIP(), err.Error()))
		}
		c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=fail"))
		return
	}

	topUp := model.GetTopUpByTradeNo(authority)
	if topUp == nil {
		logger.LogWarn(c.Request.Context(), common.LogText("ZarinPal callback order not found trade_no=%s client_ip=%s", authority, c.ClientIP()))
		c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=fail"))
		return
	}
	if topUp.Status == common.TopUpStatusSuccess {
		logger.LogInfo(c.Request.Context(), common.LogText("ZarinPal callback duplicate redirect ignored trade_no=%s client_ip=%s", authority, c.ClientIP()))
		c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=success"))
		return
	}
	if topUp.Status != common.TopUpStatusPending {
		logger.LogWarn(c.Request.Context(), common.LogText("ZarinPal callback order status invalid trade_no=%s status=%s client_ip=%s", authority, topUp.Status, c.ClientIP()))
		c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=fail"))
		return
	}

	verification, err := GetZarinpalClient().Verify(c.Request.Context(), authority, setting.ZarinpalRialAmount(topUp.Money))
	if err != nil {
		// The gateway may still have captured the money, so the order stays
		// pending for the administrator to reconcile or complete manually.
		logger.LogWarn(c.Request.Context(), common.LogText("ZarinPal verification failed trade_no=%s user_id=%d client_ip=%s error=%q", authority, topUp.UserId, c.ClientIP(), err.Error()))
		c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=pending"))
		return
	}

	LockOrder(authority)
	defer UnlockOrder(authority)
	alreadyDone, err := model.RechargeZarinpal(authority, c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, model.ErrTopUpNotFound):
			logger.LogWarn(c.Request.Context(), common.LogText("ZarinPal top-up order not found trade_no=%s client_ip=%s", authority, c.ClientIP()))
		case errors.Is(err, model.ErrPaymentMethodMismatch):
			logger.LogWarn(c.Request.Context(), common.LogText("ZarinPal order payment gateway mismatch trade_no=%s client_ip=%s", authority, c.ClientIP()))
		case errors.Is(err, model.ErrTopUpStatusInvalid):
			logger.LogWarn(c.Request.Context(), common.LogText("ZarinPal order status invalid trade_no=%s client_ip=%s", authority, c.ClientIP()))
		default:
			logger.LogError(c.Request.Context(), common.LogText("ZarinPal top-up processing failed trade_no=%s client_ip=%s error=%q", authority, c.ClientIP(), err.Error()))
		}
		c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=pending"))
		return
	}

	if verification.AlreadyVerified {
		logger.LogInfo(c.Request.Context(), common.LogText("ZarinPal payment verified earlier trade_no=%s ref_id=%d already_done=%v client_ip=%s", authority, verification.RefID, alreadyDone, c.ClientIP()))
	} else {
		logger.LogInfo(c.Request.Context(), common.LogText("ZarinPal top-up settled trade_no=%s ref_id=%d fee=%.2f already_done=%v client_ip=%s", authority, verification.RefID, verification.Fee, alreadyDone, c.ClientIP()))
	}
	c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=success"))
}
